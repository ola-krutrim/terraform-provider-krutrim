package resources

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
	"github.com/ola-krutrim/krutrim-go-sdk/packages/param"
)

var _ resource.Resource = &KcmCertResource{}
var _ resource.ResourceWithConfigure = &KcmCertResource{}

type KcmCertResource struct {
	client *krutrim.Client
}

func NewKcmCertResource() resource.Resource {
	return &KcmCertResource{}
}

type KcmCertModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	VpcID      types.String `tfsdk:"vpc_id"`
	FilePath   types.String `tfsdk:"file_path"`
	Tags       types.Map    `tfsdk:"tags"`
	KRN        types.String `tfsdk:"krn"`
	Expiration types.String `tfsdk:"expiration"`
}

func (r *KcmCertResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kcm_cert"
}

func (r *KcmCertResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages KCM Certificates",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true},
			"name":       schema.StringAttribute{Required: true},
			"vpc_id":     schema.StringAttribute{Required: true},
			"file_path":  schema.StringAttribute{Required: true},
			"tags":       schema.MapAttribute{Optional: true, ElementType: types.StringType},
			"krn":        schema.StringAttribute{Computed: true},
			"expiration": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *KcmCertResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*krutrim.Client)
}

func (r *KcmCertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KcmCertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	file, err := os.Open(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File error", err.Error())
		return
	}
	defer file.Close()

	// Import the certificate
	importParams := krutrim.KcmV1CertImportParams{
		Name:     plan.Name.ValueString(),
		XVpcID:   plan.VpcID.ValueString(),
		CertFile: file,
	}

	if err = r.client.Kcm.V1.Certs.Import(ctx, importParams); err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}

	// List by VpcID to retrieve the cert details after import.
	// The API returns a single KcmV1CertGetResponse — no cert ID is exposed.
	// We use KRN as the unique resource ID.
	listParams := krutrim.KcmV1CertListParams{
		VpcID: param.Opt[string]{Value: plan.VpcID.ValueString()},
	}

	listRes, err := r.client.Kcm.V1.Certs.List(ctx, listParams)
	if err != nil {
		resp.Diagnostics.AddError("List failed", err.Error())
		return
	}

	if listRes == nil || listRes.Krn == "" {
		resp.Diagnostics.AddError("Not found", "No certificate returned after import — KRN is empty")
		return
	}

	// KRN is the only unique identifier the API returns; use it as the resource ID
	plan.ID = types.StringValue(listRes.Krn)
	plan.KRN = types.StringValue(listRes.Krn)
	plan.Expiration = types.StringValue(listRes.Expiration)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KcmCertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KcmCertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List using VpcID to refresh state.
	// The SDK Get() requires a certId which the API never returns,
	// so we use List() filtered by VpcID instead.
	listParams := krutrim.KcmV1CertListParams{
		VpcID: param.Opt[string]{Value: state.VpcID.ValueString()},
	}

	res, err := r.client.Kcm.V1.Certs.List(ctx, listParams)
	if err != nil {
		// Assume gone
		resp.State.RemoveResource(ctx)
		return
	}

	if res == nil || res.Krn == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	state.KRN = types.StringValue(res.Krn)
	state.Expiration = types.StringValue(res.Expiration)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KcmCertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KcmCertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	file, err := os.Open(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File error", err.Error())
		return
	}
	defer file.Close()

	// Update requires a certId path param — use the KRN stored as ID
	// (adjust if the API actually expects a different identifier)
	updateParams := krutrim.KcmV1CertUpdateParams{
		XVpcID:   plan.VpcID.ValueString(),
		CertFile: file,
	}

	if err = r.client.Kcm.V1.Certs.Update(ctx, plan.ID.ValueString(), updateParams); err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	// Refresh state after update
	listParams := krutrim.KcmV1CertListParams{
		VpcID: param.Opt[string]{Value: plan.VpcID.ValueString()},
	}

	res, err := r.client.Kcm.V1.Certs.List(ctx, listParams)
	if err != nil {
		resp.Diagnostics.AddError("List after update failed", err.Error())
		return
	}

	if res != nil && res.Krn != "" {
		plan.KRN = types.StringValue(res.Krn)
		plan.Expiration = types.StringValue(res.Expiration)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KcmCertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KcmCertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete requires certId — we stored KRN as ID since it's the only
	// unique identifier the API returns. Adjust if API expects something else.
	if err := r.client.Kcm.V1.Certs.Delete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}