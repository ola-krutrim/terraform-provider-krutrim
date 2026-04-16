package resources

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var _ resource.Resource = &AccessKeyResource{}
var _ resource.ResourceWithConfigure = &AccessKeyResource{}

type AccessKeyResource struct {
	client *krutrim.Client
}

func NewAccessKeyResource() resource.Resource {
	return &AccessKeyResource{}
}

type AccessKeyModel struct {
	ID types.String `tfsdk:"id"`

	Region types.String `tfsdk:"region"`
	Tier   types.String `tfsdk:"tier"`

	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

func (r *AccessKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kos_access_key"
}

func (r *AccessKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},

			"region": schema.StringAttribute{
				Required: true,

			},

			"tier": schema.StringAttribute{
				Required: true,
			},

			"access_key": schema.StringAttribute{
				Computed: true,
			},

			"secret_key": schema.StringAttribute{
				Computed:  true,
			},
		},
	}
}

func (r *AccessKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*krutrim.Client)
}

//
// ✅ CREATE (FIXED VERSION)
//
func (r *AccessKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessKeyModel

	// Read plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	result, err := r.client.Ko.V1.AccessKeys.Create(
		ctx,
		plan.Region.ValueString(),
		plan.Tier.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}

	// Debug
	fmt.Printf("FULL RESPONSE: %+v\n", result)

	// Extract values
	accessKey := result["access_key"]
	secretKey := result["secret_key"]

	if accessKey == "" {
		resp.Diagnostics.AddError("Missing field", "access_key not found")
		return
	}

	if secretKey == "" {
		resp.Diagnostics.AddError("Missing field", "secret_key not found")
		return
	}

	// Set state values
	plan.ID = types.StringValue(accessKey)
	plan.AccessKey = types.StringValue(accessKey)
	plan.SecretKey = types.StringValue(secretKey)

	// ✅ Correct way to set state
	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

//
// ✅ READ
//
func (r *AccessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &state)
}

//
// ✅ UPDATE (NO-OP, FORCE RECREATE)
//
func (r *AccessKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update supported
}

//
// ✅ DELETE
//
func (r *AccessKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Ko.V1.AccessKeys.Delete(
		ctx,
		state.AccessKey.ValueString(),
		state.Region.ValueString(),
		state.Tier.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}