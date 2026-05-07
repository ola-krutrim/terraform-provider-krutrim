package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

type IAMUserResource struct {
	client krutrim.IAMClient
}

type IAMUserModel struct {
	ID            types.String `tfsdk:"id"`
	UserKRN       types.String `tfsdk:"user_krn"`
	UserName      types.String `tfsdk:"user_name"`
	Email         types.String `tfsdk:"email"`
	Password      types.String `tfsdk:"password"`
	ConsoleAccess types.Bool   `tfsdk:"console_access"`
}

func NewIAMUserResource() resource.Resource {
	return &IAMUserResource{}
}

func (r *IAMUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_user"
}

func (r *IAMUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"user_krn":       schema.StringAttribute{Computed: true},
			"user_name":      schema.StringAttribute{Required: true},
			"email":          schema.StringAttribute{Required: true},
			"password":       schema.StringAttribute{Required: true},
			"console_access": schema.BoolAttribute{Required: true},
		},
	}
}

func (r *IAMUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IAMUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IAMUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := krutrim.CreateUserParams{}
	params.User.UserName = plan.UserName.ValueString()
	params.User.Email = plan.Email.ValueString()
	params.User.Password = plan.Password.ValueString()
	params.User.ConsoleAccess = plan.ConsoleAccess.ValueBool()

	apiResp, err := r.client.CreateUser(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Create user failed", err.Error())
		return
	}

	tflog.Info(ctx, "Create User Response", map[string]interface{}{"resp": apiResp})

	// API response is a flat object:
	// { "krn": "kcs::...", "username": "monissa", "email": "...", ... }
	krn, ok := apiResp["krn"].(string)
	if !ok || krn == "" {
		resp.Diagnostics.AddError("Invalid API response", "missing or invalid 'krn' in response")
		return
	}

	// Use the KRN as both the Terraform ID and user_krn
	plan.ID = types.StringValue(krn)
	plan.UserKRN = types.StringValue(krn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IAMUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IAMUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// No GET-by-ID API available; preserve existing state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IAMUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddWarning(
		"Update not supported",
		"IAM User cannot be updated in-place; destroy and recreate instead.",
	)
}

func (r *IAMUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IAMUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteUser(ctx, krutrim.DeleteUserParams{
		UserKRN: state.UserKRN.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Delete user failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}