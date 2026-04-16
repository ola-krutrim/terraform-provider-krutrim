package resources

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var (
	_ resource.Resource                = &KOSBucketResource{}
	_ resource.ResourceWithConfigure   = &KOSBucketResource{}
	_ resource.ResourceWithImportState = &KOSBucketResource{}
)

type KOSBucketResource struct {
	client *krutrim.Client
}

func NewKOSBucketResource() resource.Resource {
	return &KOSBucketResource{}
}

/* ================= METADATA ================= */

func (r *KOSBucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kos_bucket"
}

/* ================= MODEL ================= */

type KOSBucketModel struct {
	ID types.String `tfsdk:"id"`

	Name   types.String `tfsdk:"name"`
	Region types.String `tfsdk:"region"`
	Tier   types.String `tfsdk:"tier"`

	Description     types.String `tfsdk:"description"`
	Versioning      types.Bool   `tfsdk:"versioning"`
	AnonymousAccess types.Bool   `tfsdk:"anonymous_access"`

	Tags types.Map `tfsdk:"tags"`
}

/* ================= SCHEMA ================= */

func (r *KOSBucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"region": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"tier": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},

			"versioning": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},

			"anonymous_access": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},

			"tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

/* ================= CONFIGURE ================= */

func (r *KOSBucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			"Expected *krutrim.Client",
		)
		return
	}

	r.client = client
}

/* ================= CREATE ================= */

func (r *KOSBucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	// ✅ Ensure client exists
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client not initialized",
			"Provider client is nil. Check provider configuration.",
		)
		return
	}

	var plan KOSBucketModel

	// ✅ Read plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ✅ Build params
	params := krutrim.BucketCreateParams{
		XRegion:         plan.Region.ValueString(),
		XTier:           plan.Tier.ValueString(),
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		Versioning:      plan.Versioning.ValueBool(),
		AnonymousAccess: plan.AnonymousAccess.ValueBool(),
		Tags:            convertTags(plan.Tags),
	}

	// ✅ API call
	res, err := r.client.Ko.V1.Buckets.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}

	// ✅ Set state
	plan.ID = types.StringValue(res.KRN)

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

/* ================= READ ================= */

func (r *KOSBucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KOSBucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	_, err := r.client.Ko.V1.Buckets.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.State.Set(ctx, &state)
}

/* ================= DELETE ================= */

func (r *KOSBucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KOSBucketModel
	req.State.Get(ctx, &state)

	err := r.client.Ko.V1.Buckets.Delete(ctx, state.ID.ValueString(), state.Region.ValueString(), state.Tier.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}

/* ================= IMPORT ================= */

func (r *KOSBucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	resp.State.SetAttribute(ctx, path.Root("region"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("id"), parts[1])
}
func (r *KOSBucketResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan KOSBucketModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	params := krutrim.BucketUpdateParams{
		XRegion:         plan.Region.ValueString(),
		XTier:           plan.Tier.ValueString(),
		BucketKRN:       plan.ID.ValueString(),
		Versioning:      plan.Versioning.ValueBool(),
		AnonymousAccess: plan.AnonymousAccess.ValueBool(),
		Tags:            convertTags(plan.Tags),
	}

	_, err := r.client.Ko.V1.Buckets.Update(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	resp.State.Set(ctx, &plan)
}


func convertTags(tags types.Map) map[string]string {
	result := make(map[string]string)

	if tags.IsNull() || tags.IsUnknown() {
		return result
	}

	for k, v := range tags.Elements() {
		result[k] = v.(types.String).ValueString()
	}

	return result
}