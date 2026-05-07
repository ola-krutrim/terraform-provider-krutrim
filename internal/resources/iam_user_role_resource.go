package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

type IAMUserRoleBindingResource struct {
	client krutrim.IAMClient
}

type IAMUserRoleBindingModel struct {
	ID      types.String `tfsdk:"id"`
	UserID  types.String `tfsdk:"user_id"`
	RoleIDs types.List   `tfsdk:"role_ids"`
}

func NewIAMUserRoleBindingResource() resource.Resource {
	return &IAMUserRoleBindingResource{}
}

func (r *IAMUserRoleBindingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_user_role_binding"
}

func (r *IAMUserRoleBindingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Computed: true},
			"user_id":  schema.StringAttribute{Required: true},
			"role_ids": schema.ListAttribute{ElementType: types.StringType, Required: true},
		},
	}
}

func (r *IAMUserRoleBindingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IAMUserRoleBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IAMUserRoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roleIDs []string
	for _, v := range plan.RoleIDs.Elements() {
		roleIDs = append(roleIDs, v.(types.String).ValueString())
	}

	// API: PUT /iam/v1/urgp/user/role/group
	// Body: { "userId": "...", "roleIds": [...], "groupIds": [] }
	// groupIds must always be sent (even as empty slice) per API contract.
	_, err := r.client.AssignRolesToUser(ctx, krutrim.AssignRolesParams{
		UserID:   plan.UserID.ValueString(),
		RoleIDs:  roleIDs,
		GroupIDs: []string{}, // required by API even when empty
	})
	if err != nil {
		resp.Diagnostics.AddError("Assign roles to user failed", err.Error())
		return
	}

	plan.ID = plan.UserID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IAMUserRoleBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IAMUserRoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// No read API available — preserve existing state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IAMUserRoleBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The underlying API is a full replacement (PUT), so Update == Create.
	var plan IAMUserRoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roleIDs []string
	for _, v := range plan.RoleIDs.Elements() {
		roleIDs = append(roleIDs, v.(types.String).ValueString())
	}

	_, err := r.client.AssignRolesToUser(ctx, krutrim.AssignRolesParams{
		UserID:   plan.UserID.ValueString(),
		RoleIDs:  roleIDs,
		GroupIDs: []string{},
	})
	if err != nil {
		resp.Diagnostics.AddError("Update user role binding failed", err.Error())
		return
	}

	plan.ID = plan.UserID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IAMUserRoleBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// To "delete" a binding, replace the user's roles with an empty list.
	var state IAMUserRoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.AssignRolesToUser(ctx, krutrim.AssignRolesParams{
		UserID:   state.UserID.ValueString(),
		RoleIDs:  []string{},
		GroupIDs: []string{},
	})
	if err != nil {
		resp.Diagnostics.AddError("Remove user role binding failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}