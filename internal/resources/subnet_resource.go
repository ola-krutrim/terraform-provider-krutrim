package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-silicon/krutrim-go-sdk"
	"github.com/ola-silicon/krutrim-go-sdk/option"
)

var (
	_ resource.Resource                = &SubnetResource{}
	_ resource.ResourceWithConfigure   = &SubnetResource{}
	_ resource.ResourceWithImportState = &SubnetResource{}
)

type SubnetResource struct {
	client *krutrim.Client
}

type SubnetModel struct {
	ID        types.String `tfsdk:"id"`
	Region    types.String `tfsdk:"region"`
	VpcID     types.String `tfsdk:"vpc_id"`
	NetworkID types.String `tfsdk:"network_id"`

	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CIDR        types.String `tfsdk:"cidr"`
	GatewayIP   types.String `tfsdk:"gateway_ip"`
	IPVersion   types.String `tfsdk:"ip_version"`
	Ingress     types.Bool   `tfsdk:"ingress"`
	Egress      types.Bool   `tfsdk:"egress"`
}

func NewSubnetResource() resource.Resource {
	return &SubnetResource{}
}

func (r *SubnetResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (r *SubnetResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Subnet resource",

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

			"vpc_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"network_id": schema.StringAttribute{
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

			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"cidr": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"gateway_ip": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"ip_version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("4"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"ingress": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"egress": schema.BoolAttribute{
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

func (r *SubnetResource) Configure(
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
// CREATE
//

func (r *SubnetResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {

	var plan SubnetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := krutrim.CreateSubnetNewParams{
		VpcID: plan.VpcID.ValueString(),
		SubnetData: krutrim.SubnetData{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			CIDR:        plan.CIDR.ValueString(),
			GatewayIP:   plan.GatewayIP.ValueString(),
			NetworkID:   plan.NetworkID.ValueString(),
			IPVersion:   plan.IPVersion.ValueString(),
			Ingress:     plan.Ingress.ValueBool(),
			Egress:      plan.Egress.ValueBool(),
		},
	}

	var httpResp *http.Response

	err := r.client.CreateSubnet.New(
		ctx,
		params,
		option.WithHeader("x-region", plan.Region.ValueString()),
		option.WithResponseInto(&httpResp),
	)
	if err != nil {
		resp.Diagnostics.AddError("Create Failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	var result struct {
		SubnetID string `json:"subnet_id"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Failed", string(body))
		return
	}

	if result.SubnetID == "" {
		resp.Diagnostics.AddError("Create Failed", "No subnet_id returned")
		return
	}

	plan.ID = types.StringValue(result.SubnetID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

//
// READ
//

func (r *SubnetResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SubnetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Optional: you can implement SearchSubnet here
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

//
// DELETE
//
func (r *SubnetResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SubnetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSubnet.Delete(
		ctx,
		krutrim.DeleteSubnetDeleteParams{
			SubnetID: state.ID.ValueString(),
			VpcID:    state.VpcID.ValueString(),
			XRegion:  state.Region.ValueString(),
		},
	)

	if err != nil {
		errMsg := strings.ToLower(err.Error())


		if strings.Contains(errMsg, "primary subnet") ||
			strings.Contains(errMsg, "cannot delete this subnet individually") {

			tflog.Warn(ctx, "Primary subnet cannot be deleted independently; removing from state", map[string]interface{}{
				"subnet_id": state.ID.ValueString(),
				"vpc_id":    state.VpcID.ValueString(),
			})

			// IMPORTANT: remove from Terraform state anyway
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Delete Failed", err.Error())
		return
	}


	resp.State.RemoveResource(ctx)
}

//
// IMPORT
//

func (r *SubnetResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {

	// region:vpc_id:subnet_id
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: region:vpc_id:subnet_id",
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("region"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("vpc_id"), parts[1])
	resp.State.SetAttribute(ctx, path.Root("id"), parts[2])
}
//
// UPDATE
//

func (r *SubnetResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	// All fields use RequiresReplace, so Update should never be called.
	// This method exists only to satisfy the Terraform interface.

	var plan SubnetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
