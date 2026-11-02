package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var (
	_ datasource.DataSource              = &KKSFlavorsDataSource{}
	_ datasource.DataSourceWithConfigure = &KKSFlavorsDataSource{}
)

type KKSFlavorsDataSource struct {
	client *krutrim.Client
}

type KKSFlavorModel struct {
	Name      types.String `tfsdk:"name"`
	CPU       types.Int64  `tfsdk:"cpu"`
	Memory    types.String `tfsdk:"memory"`
	GPUCount  types.Int64  `tfsdk:"gpu_count"`
	GPUMemory types.Int64  `tfsdk:"gpu_memory"`
}

type KKSFlavorsDataSourceModel struct {
	Region  types.String    `tfsdk:"region"`
	Flavors []KKSFlavorModel `tfsdk:"flavors"`
}

func NewKKSFlavorsDataSource() datasource.DataSource {
	return &KKSFlavorsDataSource{}
}

func (d *KKSFlavorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kks_flavors"
}

func (d *KKSFlavorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists instance flavors available for KKS node groups.",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Required: true,
			},
			"flavors": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":       schema.StringAttribute{Computed: true},
						"cpu":        schema.Int64Attribute{Computed: true},
						"memory":     schema.StringAttribute{Computed: true},
						"gpu_count":  schema.Int64Attribute{Computed: true},
						"gpu_memory": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *KKSFlavorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KKSFlavorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KKSFlavorsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flavors, err := d.client.KKS.V1.Flavor.List(ctx, krutrim.KKSV1FlavorListParams{
		XRegion: config.Region.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("List KKS Flavors Failed", err.Error())
		return
	}

	state := KKSFlavorsDataSourceModel{
		Region:  config.Region,
		Flavors: make([]KKSFlavorModel, 0, len(flavors)),
	}
	for _, f := range flavors {
		state.Flavors = append(state.Flavors, KKSFlavorModel{
			Name:      types.StringValue(f.Name),
			CPU:       types.Int64Value(f.CPU),
			Memory:    types.StringValue(f.Memory),
			GPUCount:  types.Int64Value(f.GPUCount),
			GPUMemory: types.Int64Value(f.GPUMemory),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
