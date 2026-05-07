package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

type IAMProgrammaticAccessResource struct {
	client krutrim.IAMClient
}

type IAMProgrammaticAccessModel struct {
	ID        types.String `tfsdk:"id"`
	UserKRN   types.String `tfsdk:"user_krn"`
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

func NewIAMProgrammaticAccessResource() resource.Resource {
	return &IAMProgrammaticAccessResource{}
}

func (r *IAMProgrammaticAccessResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_programmatic_access"
}

func (r *IAMProgrammaticAccessResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true},
			"user_krn":   schema.StringAttribute{Required: true},
			"access_key": schema.StringAttribute{Computed: true},
			"secret_key": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *IAMProgrammaticAccessResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IAMProgrammaticAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IAMProgrammaticAccessModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.EnableProgrammaticAccess(ctx, krutrim.EnableProgrammaticAccessParams{
		UserKRN: plan.UserKRN.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Enable programmatic access failed", err.Error())
		return
	}

	// API response structure:
	// {
	//   "message": "Programmatic access is enabled for the user",
	//   "programmaticAccessInfo": {
	//     "accessKey": "kpa...",
	//     "secret":    "xTO...",       // NOTE: key is "secret", not "secretKey"
	//     "programmaticAccessEnabled": true
	//   }
	// }
	accessInfo, ok := apiResp["programmaticAccessInfo"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Invalid API response", "missing 'programmaticAccessInfo' in response")
		return
	}

	accessKey, ok := accessInfo["accessKey"].(string)
	if !ok || accessKey == "" {
		resp.Diagnostics.AddError("Invalid API response", "missing or invalid 'accessKey'")
		return
	}

	// The API returns the secret under the key "secret", not "secretKey"
	secret, ok := accessInfo["secret"].(string)
	if !ok || secret == "" {
		resp.Diagnostics.AddError("Invalid API response", "missing or invalid 'secret'")
		return
	}

	plan.AccessKey = types.StringValue(accessKey)
	plan.SecretKey = types.StringValue(secret)
	plan.ID = plan.UserKRN

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IAMProgrammaticAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IAMProgrammaticAccessModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// Credentials are write-once; no read API — preserve existing state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IAMProgrammaticAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddWarning(
		"Update not supported",
		"Programmatic access credentials cannot be updated; destroy and recreate instead.",
	)
}

func (r *IAMProgrammaticAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No explicit disable API — remove from Terraform state only.
	resp.State.RemoveResource(ctx)
}