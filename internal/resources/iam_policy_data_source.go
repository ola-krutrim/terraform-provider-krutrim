package resources

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

type IAMPoliciesDataSource struct {
	client krutrim.IAMClient
}

type IAMPoliciesModel struct {
	Policies []PolicyModel `tfsdk:"policies"`
}

type PolicyModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewIAMPoliciesDataSource() datasource.DataSource {
	return &IAMPoliciesDataSource{}
}

func (d *IAMPoliciesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_policies"
}

func (d *IAMPoliciesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"policies": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *IAMPoliciesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	mainClient, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data Type", "Expected *krutrim.Client")
		return
	}

	d.client = krutrim.NewIAMClient(mainClient.Options...)
}

func (d *IAMPoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IAMPoliciesModel

	apiResp, err := d.client.ListPolicies(ctx, krutrim.ListPoliciesParams{
		Limit:          100,
		Offset:         0,
		KrutrimManaged: "all",
	})
	if err != nil {
		resp.Diagnostics.AddError("List policies failed", err.Error())
		return
	}

	policiesRaw, ok := apiResp["policies"].([]interface{})
	if !ok {
		resp.Diagnostics.AddError("Invalid API response", "missing policies")
		return
	}

	for _, p := range policiesRaw {
		pMap, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := pMap["_id"].(string)   // 🔥 IMPORTANT CHANGE
		name, _ := pMap["name"].(string)

		state.Policies = append(state.Policies, PolicyModel{
			ID:   types.StringValue(id),
			Name: types.StringValue(name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}