package resources

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ola-krutrim/krutrim-go-sdk/packages/param"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var _ resource.Resource = &KcmCertResource{}
var _ resource.ResourceWithConfigure = &KcmCertResource{}

type KcmCertResource struct {
	client *krutrim.Client
}

func NewKcmCertResource() resource.Resource {
	return &KcmCertResource{}
}

//
// ========================
// MODEL
// ========================
//

type KcmCertModel struct {
	ID types.String `tfsdk:"id"`

	Name     types.String `tfsdk:"name"`
	VpcID    types.String `tfsdk:"vpc_id"`
	FilePath types.String `tfsdk:"file_path"`

	Tags types.Map `tfsdk:"tags"`

	KRN        types.String `tfsdk:"krn"`
	Expiration types.String `tfsdk:"expiration"`
}

//
// ========================
// METADATA
// ========================
//

func (r *KcmCertResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kcm_cert"
}

//
// ========================
// SCHEMA
// ========================
//

func (r *KcmCertResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{

		MarkdownDescription: "Manages KCM Certificates",

		Attributes: map[string]schema.Attribute{

			"id": schema.StringAttribute{
				Computed: true,
			},

			"name": schema.StringAttribute{
				Required: true,
			},

			"vpc_id": schema.StringAttribute{
				Required: true,
			},

			"file_path": schema.StringAttribute{
				Required: true,
			},

			"tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},

			"krn": schema.StringAttribute{
				Computed: true,
			},

			"expiration": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

//
// ========================
// CONFIGURE
// ========================
//

func (r *KcmCertResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*krutrim.Client)
}

//
// ========================
// CREATE
// ========================
//

func (r *KcmCertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KcmCertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	file, err := os.Open(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File error", err.Error())
		return
	}
	defer file.Close()

	// STEP 1: Import
	params := krutrim.KcmV1CertImportParams{
		Name:     plan.Name.ValueString(),
		XVpcID:   plan.VpcID.ValueString(),
		CertFile: file,
	}

	err = r.client.Kcm.V1.Certs.Import(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}

	// STEP 2: Retry LIST to get certID
	var certID string
	var listRes *krutrim.KcmV1CertListResponse // ✅ ADD THIS
	
	for i := 0; i < 5; i++ {
		listRes, err = r.client.Kcm.V1.Certs.List(ctx, krutrim.KcmV1CertListParams{
			VpcID: param.NewOpt(plan.VpcID.ValueString()),
		})
		if err != nil {
			resp.Diagnostics.AddError("List failed", err.Error())
			return
		}

		for _, c := range listRes.Certificates {
			if c.CertName == plan.Name.ValueString() {
				certID = c.CertID
				break
			}
		}

		if certID != "" {
			break
		}

		time.Sleep(2 * time.Second)
	}

	if certID == "" {
		resp.Diagnostics.AddError("Cert not found", "Unable to find created certificate after retries")
		return
	}

	// STEP 3: Set ID
	plan.ID = types.StringValue(certID)

	plan.KRN = types.StringValue(certID)

	for _, c := range listRes.Certificates {
		if c.CertID == certID {
			plan.Expiration = types.StringValue(c.ExpirationDate)
			break
		}
	}

	// STEP 5: Tags
	if !plan.Tags.IsNull() {
		tagMap := map[string]string{}
		for k, v := range plan.Tags.Elements() {
			tagMap[k] = v.(types.String).ValueString()
		}

		tagParams := krutrim.KcmV1CertTagAddParams{
			Body: tagMap,
		}

		_ = r.client.Kcm.V1.Certs.Tags.Add(ctx, certID, tagParams)
	}

	resp.State.Set(ctx, &plan)
}

//
// ========================
// READ
// ========================
//

func (r *KcmCertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KcmCertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	listRes, err := r.client.Kcm.V1.Certs.List(ctx, krutrim.KcmV1CertListParams{
		VpcID: param.NewOpt(state.VpcID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Read failed", err.Error())
		return
	}

	found := false

	for _, c := range listRes.Certificates {
		if c.CertID == state.ID.ValueString() {
			state.KRN = types.StringValue(c.CertID)
			state.Expiration = types.StringValue(c.ExpirationDate)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.State.Set(ctx, &state)
}

//
// ========================
// UPDATE
// ========================
//

func (r *KcmCertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KcmCertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	file, err := os.Open(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File error", err.Error())
		return
	}
	defer file.Close()

	params := krutrim.KcmV1CertUpdateParams{
		XVpcID:   plan.VpcID.ValueString(),
		CertFile: file,
	}

	err = r.client.Kcm.V1.Certs.Update(ctx, plan.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	listRes, err := r.client.Kcm.V1.Certs.List(ctx, krutrim.KcmV1CertListParams{
		VpcID: param.NewOpt(plan.VpcID.ValueString()),
	})
	
	if err == nil {
		for _, c := range listRes.Certificates {
			if c.CertID == plan.ID.ValueString() {
				plan.KRN = types.StringValue(c.CertID)
				plan.Expiration = types.StringValue(c.ExpirationDate)
				break
			}
		}
	}

	resp.State.Set(ctx, &plan)
}

//
// ========================
// DELETE
// ========================
//

func (r *KcmCertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KcmCertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	err := r.client.Kcm.V1.Certs.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}