package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var (
	_ datasource.DataSource              = &KKSAddonsDataSource{}
	_ datasource.DataSourceWithConfigure = &KKSAddonsDataSource{}
)

type KKSAddonsDataSource struct {
	client *krutrim.Client
}

type KKSAddonCompatibilityModel struct {
	ClusterVersion   types.String `tfsdk:"cluster_version"`
	PlatformVersions types.List   `tfsdk:"platform_versions"`
	DefaultVersion   types.Bool   `tfsdk:"default_version"`
}

type KKSAddonVersionModel struct {
	AddonVersion    types.String                 `tfsdk:"addon_version"`
	Architecture    types.List                   `tfsdk:"architecture"`
	Compatibilities []KKSAddonCompatibilityModel `tfsdk:"compatibilities"`
}

type KKSAddonCatalogModel struct {
	AddonName     types.String           `tfsdk:"addon_name"`
	Type          types.String           `tfsdk:"type"`
	Description   types.String           `tfsdk:"description"`
	IsDeletable   types.Bool             `tfsdk:"is_deletable"`
	AddonVersions []KKSAddonVersionModel `tfsdk:"addon_versions"`
}

type KKSAddonsDataSourceModel struct {
	Region types.String           `tfsdk:"region"`
	Addons []KKSAddonCatalogModel `tfsdk:"addons"`
}

func NewKKSAddonsDataSource() datasource.DataSource {
	return &KKSAddonsDataSource{}
}

func (d *KKSAddonsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kks_addons"
}

func (d *KKSAddonsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists add-ons available for installation on KKS clusters.",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Required: true,
			},
			"addons": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"addon_name":   schema.StringAttribute{Computed: true},
						"type":         schema.StringAttribute{Computed: true},
						"description":  schema.StringAttribute{Computed: true},
						"is_deletable": schema.BoolAttribute{Computed: true},
						"addon_versions": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"addon_version": schema.StringAttribute{Computed: true},
									"architecture": schema.ListAttribute{
										Computed:    true,
										ElementType: types.StringType,
									},
									"compatibilities": schema.ListNestedAttribute{
										Computed: true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"cluster_version": schema.StringAttribute{Computed: true},
												"platform_versions": schema.ListAttribute{
													Computed:    true,
													ElementType: types.StringType,
												},
												"default_version": schema.BoolAttribute{Computed: true},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *KKSAddonsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *krutrim.Client, got %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *KKSAddonsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KKSAddonsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listed, err := d.client.KKS.V1.Addon.List(ctx, krutrim.KKSV1AddonListParams{
		XRegion: config.Region.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("List KKS Addons Failed", err.Error())
		return
	}

	state := KKSAddonsDataSourceModel{
		Region: config.Region,
		Addons: make([]KKSAddonCatalogModel, 0, len(listed.Addons)),
	}

	for _, addon := range listed.Addons {
		versions := make([]KKSAddonVersionModel, 0, len(addon.AddonVersions))
		for _, ver := range addon.AddonVersions {
			archValues := make([]attr.Value, 0, len(ver.Architecture))
			for _, a := range ver.Architecture {
				archValues = append(archValues, types.StringValue(a))
			}

			comps := make([]KKSAddonCompatibilityModel, 0, len(ver.Compatibilities))
			for _, c := range ver.Compatibilities {
				platValues := make([]attr.Value, 0, len(c.PlatformVersions))
				for _, p := range c.PlatformVersions {
					platValues = append(platValues, types.StringValue(p))
				}
				comps = append(comps, KKSAddonCompatibilityModel{
					ClusterVersion:   types.StringValue(c.ClusterVersion),
					PlatformVersions: types.ListValueMust(types.StringType, platValues),
					DefaultVersion:   types.BoolValue(c.DefaultVersion),
				})
			}

			versions = append(versions, KKSAddonVersionModel{
				AddonVersion:    types.StringValue(ver.AddonVersion),
				Architecture:    types.ListValueMust(types.StringType, archValues),
				Compatibilities: comps,
			})
		}

		state.Addons = append(state.Addons, KKSAddonCatalogModel{
			AddonName:     types.StringValue(addon.AddonName),
			Type:          types.StringValue(addon.Type),
			Description:   types.StringValue(addon.Description),
			IsDeletable:   types.BoolValue(addon.IsDeletable),
			AddonVersions: versions,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
