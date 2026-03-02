package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-silicon/krutrim-go-sdk"
	"github.com/ola-silicon/krutrim-go-sdk/option"
)

var (
	_ resource.Resource                = &FloatingIPResource{}
	_ resource.ResourceWithConfigure   = &FloatingIPResource{}
	_ resource.ResourceWithImportState = &FloatingIPResource{}
)

type FloatingIPResource struct {
	client *krutrim.Client
}

type FloatingIPModel struct {
	ID         types.String `tfsdk:"id"`
	Region     types.String `tfsdk:"region"`
	Name       types.String `tfsdk:"name"`
	VpcID      types.String `tfsdk:"vpc_id"`
	FloatingIP types.Bool   `tfsdk:"floating_ip"`
}

type CreatePortResponse struct {
	FloatingIPAddress string `json:"floating_ip_address"`
	FloatingIPKRN     string `json:"floating_ip_krn"`
	PortKRN           string `json:"port_krn"`
	Message           string `json:"message"`
}

type FloatingIP struct {
	FloatingIPKRN string `json:"floating_ip_krn"`
}

func NewFloatingIPResource() resource.Resource {
	return &FloatingIPResource{}
}

func (r *FloatingIPResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_floating_ip"
}

func (r *FloatingIPResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Floating IP resource",

		Attributes: map[string]schema.Attribute{

			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"region": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"vpc_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"floating_ip": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *FloatingIPResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			fmt.Sprintf("Expected *krutrim.Client, got %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

//
// ======================
// CREATE
// ======================
//

func (r *FloatingIPResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {

	var plan FloatingIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkID, subnetID, err := r.resolveNetworkAndSubnet(
		ctx,
		plan.VpcID.ValueString(),
		plan.Region.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve VPC details", err.Error())
		return
	}

	params := krutrim.CreatePortNewParams{
		XRegion:    plan.Region.ValueString(),
		Name:       plan.Name.ValueString(),
		VpcID:      plan.VpcID.ValueString(),
		SubnetID:   subnetID,
		NetworkID:  networkID,
		FloatingIP: plan.FloatingIP.ValueBool(),
	}

	var httpResp *http.Response

	err = r.client.CreatePort.New(
		ctx,
		params,
		option.WithResponseInto(&httpResp),
	)
	if err != nil {
		resp.Diagnostics.AddError("Create Failed", err.Error())
		return
	}

	var result CreatePortResponse
	if err := decodeResponse(httpResp, &result); err != nil {
		resp.Diagnostics.AddError("Decode Failed", err.Error())
		return
	}

	if result.FloatingIPKRN == "" {
		resp.Diagnostics.AddError("Create Failed", "API did not return floating_ip_krn")
		return
	}

	plan.ID = types.StringValue(result.FloatingIPKRN)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

//
// ======================
// READ
// ======================
//

func (r *FloatingIPResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {

	var state FloatingIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.floatingIPExists(
		ctx,
		state.ID.ValueString(),
		state.VpcID.ValueString(),
		state.Region.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddWarning("Read Failed", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
	}
}

//
// ======================
// UPDATE
// ======================
//

func (r *FloatingIPResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan FloatingIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

//
// ======================
// DELETE
// ======================
//

func (r *FloatingIPResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {

	var state FloatingIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteFloatingIP.Release(
		ctx,
		krutrim.DeleteFloatingIPReleaseParams{
			FloatingIPKrn: state.ID.ValueString(),
			XRegion:       state.Region.ValueString(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Delete Failed", err.Error())
		return
	}

	r.waitForDeletion(ctx, state.ID.ValueString(), state.VpcID.ValueString(), state.Region.ValueString())
}

//
// ======================
// IMPORT
// ======================
//

func (r *FloatingIPResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {

	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Format must be region:floating_ip_krn",
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("region"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("id"), parts[1])
}

//
// ======================
// HELPERS
// ======================
//

func decodeResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

func (r *FloatingIPResource) floatingIPExists(
	ctx context.Context,
	id string,
	vpcID string,
	region string,
) (bool, error) {

	var httpResp *http.Response

	err := r.client.FloatingIPList.List(
		ctx,
		krutrim.FloatingIPListListParams{
			VpcID:   vpcID,
			XRegion: region,
		},
		option.WithResponseInto(&httpResp),
	)
	if err != nil {
		return false, err
	}

	var raw []FloatingIP
	if err := decodeResponse(httpResp, &raw); err != nil {
		return false, err
	}

	for _, ip := range raw {
		if ip.FloatingIPKRN == id {
			return true, nil
		}
	}

	return false, nil
}

func (r *FloatingIPResource) waitForDeletion(
	ctx context.Context,
	id string,
	vpcID string,
	region string,
) {

	for i := 0; i < 20; i++ {

		exists, _ := r.floatingIPExists(ctx, id, vpcID, region)

		if !exists {
			return
		}

		time.Sleep(4 * time.Second)
	}
}

func (r *FloatingIPResource) resolveNetworkAndSubnet(
	ctx context.Context,
	vpcID string,
	region string,
) (string, string, error) {

	var httpResp *http.Response

	err := r.client.DescribeVpc.Get(
		ctx,
		krutrim.DescribeVpcGetParams{
			VpcID:   vpcID,
			XRegion: region,
		},
		option.WithResponseInto(&httpResp),
	)
	if err != nil {
		return "", "", err
	}

	var raw map[string]struct {
		Networks struct {
			KrnID string `json:"krn_id"`
		} `json:"networks"`
		Subnets []struct {
			KrnID  string `json:"krn_id"`
			Status string `json:"status"`
		} `json:"subnets"`
	}

	if err := decodeResponse(httpResp, &raw); err != nil {
		return "", "", err
	}

	for _, vpcData := range raw {
		for _, subnet := range vpcData.Subnets {
			if strings.ToUpper(subnet.Status) == "ACTIVE" {
				return vpcData.Networks.KrnID, subnet.KrnID, nil
			}
		}
	}

	return "", "", fmt.Errorf("no active subnet found")
}
