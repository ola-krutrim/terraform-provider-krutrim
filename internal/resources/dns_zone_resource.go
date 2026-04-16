package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
	"github.com/ola-krutrim/krutrim-go-sdk/option"
)

var (
	_ resource.Resource                = &DNSZoneResource{}
	_ resource.ResourceWithConfigure   = &DNSZoneResource{}
	_ resource.ResourceWithImportState = &DNSZoneResource{}
)

type DNSZoneResource struct {
	client *krutrim.Client
}

type DNSZoneModel struct {
	ID       types.String `tfsdk:"id"`
	Type     types.String `tfsdk:"type"`
	VpcID    types.String `tfsdk:"vpc_id"`
	ZoneName types.String `tfsdk:"zone_name"`
}

func NewDNSZoneResource() resource.Resource {
	return &DNSZoneResource{}
}

/* ================= METADATA ================= */

func (r *DNSZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

/* ================= SCHEMA ================= */

func (r *DNSZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages DNS Zone",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
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
			"zone_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

/* ================= CONFIGURE ================= */

func (r *DNSZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *krutrim.Client")
		return
	}

	r.client = client
}

/* ================= CREATE ================= */

func (r *DNSZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSZoneModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating DNS Zone", map[string]interface{}{
		"name": plan.ZoneName.ValueString(),
	})

	params := krutrim.DNSV1ZoneAddParams{
		Type:     plan.Type.ValueString(),
		Vpcid:    plan.VpcID.ValueString(),
		Zonename: plan.ZoneName.ValueString(),
	}

	var httpResp *http.Response

	err := r.client.DNS.V1.Zone.Add(ctx, params, option.WithResponseInto(&httpResp))
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode >= 400 {
		resp.Diagnostics.AddError("API Error", string(body))
		return
	}

	// ✅ Correct parsing
	var apiResp struct {
		Data struct {
			KRN string `json:"krn"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Parse error", err.Error())
		return
	}

	if apiResp.Data.KRN == "" {
		resp.Diagnostics.AddError("Invalid response", "Missing zone KRN")
		return
	}

	plan.ID = types.StringValue(apiResp.Data.KRN)
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)
	
		var checkResp *http.Response
		err := r.client.DNS.V1.Zone.Get(ctx, plan.ID.ValueString(), option.WithResponseInto(&checkResp))
		if err != nil {
			continue
		}
	
		if checkResp == nil {
			continue
		}
	
		body, _ := io.ReadAll(checkResp.Body)
		checkResp.Body.Close()
	
		var respData map[string]interface{}
		json.Unmarshal(body, &respData)
	
		if data, ok := respData["data"].(map[string]interface{}); ok {
			if status, ok := data["status"].(string); ok && status == "ACTIVE" {
				tflog.Info(ctx, "DNS Zone is ACTIVE and ready")
				break
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

/* ================= READ ================= */

func (r *DNSZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSZoneModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var httpResp *http.Response

	err := r.client.DNS.V1.Zone.Get(ctx, state.ID.ValueString(), option.WithResponseInto(&httpResp))
	if err != nil {
		tflog.Warn(ctx, "Read failed, keeping state", map[string]interface{}{"error": err})
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode >= 400 {
		return
	}

	body, _ := io.ReadAll(httpResp.Body)

	var apiResp struct {
		Data struct {
			ZoneName string `json:"zonename"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Parse error", err.Error())
		return
	}

	if apiResp.Data.ZoneName != "" {
		state.ZoneName = types.StringValue(apiResp.Data.ZoneName)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

/* ================= DELETE ================= */

func (r *DNSZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSZoneModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DNS.V1.Zone.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}

/* ================= IMPORT ================= */

func (r *DNSZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

/* ================= UPDATE ================= */

func (r *DNSZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSZoneModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No update API → just sync state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}