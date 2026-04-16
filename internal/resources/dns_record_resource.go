package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
	"github.com/ola-krutrim/krutrim-go-sdk/option"
	"github.com/ola-krutrim/krutrim-go-sdk/packages/param"
)

var _ resource.Resource = &DNSRecordResource{}
var _ resource.ResourceWithConfigure = &DNSRecordResource{}

type DNSRecordResource struct {
	client *krutrim.Client
}

type DNSRecordModel struct {
	ID     types.String `tfsdk:"id"`
	KrnID  types.String `tfsdk:"krn_id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	TTL    types.Int64  `tfsdk:"ttl"`
	Values types.List   `tfsdk:"values"`
}

func NewDNSRecordResource() resource.Resource {
	return &DNSRecordResource{}
}

func (r *DNSRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *DNSRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages DNS Record",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"krn_id": schema.StringAttribute{Required: true},
			"name":   schema.StringAttribute{Required: true},
			"type":   schema.StringAttribute{Required: true},
			"ttl":    schema.Int64Attribute{Required: true},
			"values": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
			},
		},
	}
}

func (r *DNSRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*krutrim.Client)
}

///////////////////////
// CREATE
///////////////////////
func (r *DNSRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	// Build values
	values := []krutrim.DNSV1RecordAddParamsRecord{}
	for _, v := range plan.Values.Elements() {
		values = append(values, krutrim.DNSV1RecordAddParamsRecord{
			Value: param.NewOpt(v.(types.String).ValueString()),
		})
	}

	params := krutrim.DNSV1RecordAddParams{
		Krnid:   plan.KrnID.ValueString(),
		Rname:   plan.Name.ValueString(),
		Type:    plan.Type.ValueString(),
		Ttl:     plan.TTL.ValueInt64(),
		Records: values,
	}

	var httpResp *http.Response

	err := r.client.DNS.V1.Record.Add(ctx, params, option.WithResponseInto(&httpResp))
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	tflog.Info(ctx, "Create DNS Record Response", map[string]interface{}{
		"body": string(body),
	})

	if httpResp.StatusCode >= 400 {
		resp.Diagnostics.AddError("API Error", string(body))
		return
	}

	// ✅ Correct parsing
	var apiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Parse error", err.Error())
		return
	}

	if apiResp.Data.ID == "" {
		resp.Diagnostics.AddError("Invalid response", "Missing record ID")
		return
	}

	plan.ID = types.StringValue(apiResp.Data.ID)

	// ✅ Optional: wait for async completion
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)

		var checkResp *http.Response
		err := r.client.DNS.V1.GetRecords(ctx, plan.KrnID.ValueString(), option.WithResponseInto(&checkResp))
		if err != nil {
			continue
		}
		defer checkResp.Body.Close()

		checkBody, _ := io.ReadAll(checkResp.Body)

		var data map[string]interface{}
		json.Unmarshal(checkBody, &data)

		dataObj, ok := data["data"].(map[string]interface{})
		if !ok {
			continue
		}

		records, ok := dataObj["records"].([]interface{})
		if !ok {
			continue
		}

		for _, rec := range records {
			rMap := rec.(map[string]interface{})
			if rMap["id"] == plan.ID.ValueString() {
				if rMap["status"] == "success" {
					break
				}
			}
		}
	}

	resp.State.Set(ctx, &plan)
}

///////////////////////
// READ
///////////////////////
func (r *DNSRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var httpResp *http.Response

	err := r.client.DNS.V1.GetRecords(ctx, state.KrnID.ValueString(), option.WithResponseInto(&httpResp))
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	body, _ := io.ReadAll(httpResp.Body)

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	// ✅ FIXED PATH
	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	rows, ok := dataObj["rows"].([]interface{}) // ✅ rows, not records
	if !ok {
		return
	}

	found := false

	for _, rec := range rows {
		rMap := rec.(map[string]interface{})

		if rMap["id"] == state.ID.ValueString() {

			// ✅ FIXED FIELD NAME
			if name, ok := rMap["name"].(string); ok {
				state.Name = types.StringValue(name)
			}

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

///////////////////////
// UPDATE
///////////////////////
func (r *DNSRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	values := []krutrim.DNSV1RecordUpdateParamsRecord{}

	for _, v := range plan.Values.Elements() {
		values = append(values, krutrim.DNSV1RecordUpdateParamsRecord{
			Value: param.NewOpt(v.(types.String).ValueString()),
		})
	}

	params := krutrim.DNSV1RecordUpdateParams{
		Rname:   param.NewOpt(plan.Name.ValueString()),
		Type:    param.NewOpt(plan.Type.ValueString()),
		Records: values,
	}

	err := r.client.DNS.V1.Record.Update(ctx, plan.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	resp.State.Set(ctx, &plan)
}

///////////////////////
// DELETE
///////////////////////
func (r *DNSRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	err := r.client.DNS.V1.Record.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}