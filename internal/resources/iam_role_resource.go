package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

type IAMRoleResource struct {
	client krutrim.IAMClient
}

type IAMRoleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PolicyIDs   types.List   `tfsdk:"policy_ids"`
}

func NewIAMRoleResource() resource.Resource {
	return &IAMRoleResource{}
}

func (r *IAMRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_role"
}

func (r *IAMRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true, Computed: true},
			"policy_ids":  schema.ListAttribute{ElementType: types.StringType, Required: true},
		},
	}
}

func (r *IAMRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	mainClient, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			"Expected *krutrim.Client",
		)
		return
	}

	r.client = krutrim.NewIAMClient(mainClient.Options...)
}

func (r *IAMRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IAMRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var policyIDs []string
	for _, v := range plan.PolicyIDs.Elements() {
		policyIDs = append(policyIDs, v.(types.String).ValueString())
	}

	description := "Created via Terraform"
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description = plan.Description.ValueString()
	}

	apiResp, err := r.client.CreateRoleWithPolicies(ctx, krutrim.CreateRoleParams{
		Role: struct {
			Name        string `json:"Name"`
			Description string `json:"Description"`
		}{
			Name:        plan.Name.ValueString(),
			Description: description,
		},
		PolicyIDs: policyIDs,
	})
	if err != nil {
		resp.Diagnostics.AddError("Create role failed", err.Error())
		return
	}

	// API response is a flat object:
	// { "krn": "kcs::...", "message": "Role Created", "policiesAttached": 1 }
	krn, ok := apiResp["krn"].(string)
	if !ok || krn == "" {
		resp.Diagnostics.AddError("Invalid API response", "missing or invalid 'krn' in response")
		return
	}

	plan.ID = types.StringValue(krn)
	plan.Description = types.StringValue(description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IAMRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IAMRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// No GET-by-ID API available; preserve existing state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IAMRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddWarning(
		"Update not supported",
		"IAM Role cannot be updated in-place; destroy and recreate instead.",
	)
}

func (r *IAMRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IAMRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteRole(ctx, krutrim.DeleteRoleParams{
		RoleKRN: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Delete role failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}