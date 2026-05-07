package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

type IAMRolesDataSource struct {
	client krutrim.IAMClient
}

type IAMRolesModel struct {
	Roles []RoleModel `tfsdk:"roles"`
}

type RoleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewIAMRolesDataSource() datasource.DataSource {
	return &IAMRolesDataSource{}
}

func (d *IAMRolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_roles"
}

func (d *IAMRolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *IAMRolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IAMRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IAMRolesModel

	apiResp, err := d.client.ListRoles(ctx, krutrim.ListRolesParams{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		resp.Diagnostics.AddError("List roles failed", err.Error())
		return
	}

	rolesRaw, ok := apiResp["roles"].([]interface{})
	if !ok {
		resp.Diagnostics.AddError("Invalid API response", "missing 'roles' in response")
		return
	}

	for _, r := range rolesRaw {
		rMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		id, _          := rMap["_id"].(string)
		name, _        := rMap["name"].(string)
		description, _ := rMap["description"].(string)

		state.Roles = append(state.Roles, RoleModel{
			ID:          types.StringValue(id),
			Name:        types.StringValue(name),
			Description: types.StringValue(description),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}