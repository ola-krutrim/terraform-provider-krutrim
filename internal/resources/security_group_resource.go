package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	krutrim "github.com/ola-silicon/krutrim-go-sdk"
	"github.com/ola-silicon/krutrim-go-sdk/packages/param"
)

type SecurityGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	VpcID       types.String `tfsdk:"vpc_id"`
	Region      types.String `tfsdk:"region"`
}

type SecurityGroupResource struct {
	client *krutrim.Client
}

func NewSecurityGroupResource() resource.Resource {
	return &SecurityGroupResource{}
}

func (r *SecurityGroupResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (r *SecurityGroupResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Manages a Krutrim Security Group",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Security Group ID (KRN)",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the security group",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the security group",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "VPC ID (KRN) where the security group belongs",
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "Region (e.g., In-Bangalore-1)",
			},
		},
	}
}

func (r *SecurityGroupResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	_ *resource.ConfigureResponse,
) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*krutrim.Client)
	}
}

func (r *SecurityGroupResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SecurityGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := krutrim.SecurityGroupV1NewParams{
		Name:    plan.Name.ValueString(),
		Vpcid:   plan.VpcID.ValueString(),
		XRegion: plan.Region.ValueString(),
	}

	if !plan.Description.IsNull() && plan.Description.ValueString() != "" {
		params.Description = param.NewOpt(plan.Description.ValueString())
	}

	sg, err := r.client.SecurityGroup.V1.New(ctx, params) 
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating security group",
			"Could not create security group: "+err.Error(),
		)
		return
	}

	if sg.ID == "" {
		raw := sg.RawJSON()

		var env map[string]any
		_ = json.Unmarshal([]byte(raw), &env)

		if res, ok := env["result"].(map[string]any); ok {
			if id, ok := res["id"].(string); ok {
				plan.ID = types.StringValue(id)
			}
		}
	} else {
		plan.ID = types.StringValue(sg.ID)
	}

	if sg.Name != "" {
		plan.Name = types.StringValue(sg.Name)
	}
	if sg.Description != "" {
		plan.Description = types.StringValue(sg.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SecurityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := krutrim.SecurityGroupV1ListParams{
		XRegion: state.Region.ValueString(),
	}

	sgList, err := r.client.SecurityGroup.V1.List(
		ctx,
		state.VpcID.ValueString(),
		params,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading security group",
			err.Error(),
		)
		return
	}

	// 🔑 FIX: parse "result" instead of "items"
	raw := sgList.RawJSON()

	var decoded struct {
		Result []krutrim.SecurityGroup `json:"result"`
	}

	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		resp.Diagnostics.AddError(
			"Error parsing security group list response",
			err.Error(),
		)
		return
	}

	var found bool
	for _, sg := range decoded.Result {
		if sg.ID == state.ID.ValueString() {
			state.Description = types.StringValue(sg.Description)
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddWarning(
			"Security Group Not Found",
			fmt.Sprintf(
				"Security group %s not found in API response. Removing from state.",
				state.ID.ValueString(),
			),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}


func (r *SecurityGroupResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"Security group updates require recreation. Use terraform apply with -replace flag.",
	)
}

func (r *SecurityGroupResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SecurityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := krutrim.SecurityGroupV1DeleteParams{
		XRegion: state.Region.ValueString(),
	}

	err := r.client.SecurityGroup.V1.Delete(ctx, state.ID.ValueString(), params) 
	if err != nil {
		errString := err.Error()
		if strings.Contains(errString, "404") || strings.Contains(errString, "not found") {
			resp.Diagnostics.AddWarning(
				"Security Group Already Deleted",
				"Security group was already deleted outside Terraform.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting security group",
			"Could not delete security group: "+err.Error(),
		)
		return
	}
}
