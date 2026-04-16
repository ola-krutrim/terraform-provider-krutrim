// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
// ⚠️ Modified for single-environment usage

package internal

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-silicon/krutrim-go-sdk"
	"github.com/ola-silicon/krutrim-go-sdk/option"

	"github.com/ola-silicon/krutrim-terraform/internal/resources"
)

// Provider implementation
type KrutrimProvider struct {
	version string
}

// Provider config model
type KrutrimProviderModel struct {
	BaseURL types.String `tfsdk:"base_url"`
	APIKey  types.String `tfsdk:"api_key"`
}

// Metadata
func (p *KrutrimProvider) Metadata(
	ctx context.Context,
	req provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "krutrim"
	resp.Version = p.version
}

// Schema
func (p *KrutrimProvider) Schema(
	ctx context.Context,
	req provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"base_url": schema.StringAttribute{
				Optional: true,
			},

			"api_key": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Configure (Create SDK Client)
func (p *KrutrimProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {

	var data KrutrimProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	opts := []option.RequestOption{}

	// Base URL (optional)
	if !data.BaseURL.IsNull() && !data.BaseURL.IsUnknown() {

		opts = append(opts,
			option.WithBaseURL(data.BaseURL.ValueString()),
		)

	} else if o, ok := os.LookupEnv("KRUTRIM_BASE_URL"); ok {

		opts = append(opts,
			option.WithBaseURL(o),
		)
	}

	// API Key (required)
	if !data.APIKey.IsNull() && !data.APIKey.IsUnknown() {

		opts = append(opts,
			option.WithAPIKey(data.APIKey.ValueString()),
		)

	} else if o, ok := os.LookupEnv("KRUTRIM_API_KEY"); ok {

		opts = append(opts,
			option.WithAPIKey(o),
		)

	} else {

		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing api_key",
			"Set api_key in provider config or KRUTRIM_API_KEY env var",
		)

		return
	}

	// Create SDK client
	client := krutrim.NewClient(opts...)

	// Pass client to resources
	resp.ResourceData = &client
}

// Resources
func (p *KrutrimProvider) Resources(
	ctx context.Context,
) []func() resource.Resource {

	return []func() resource.Resource{
		resources.NewVPCResource,
		resources.NewVolumeResource,
		resources.NewSSHKeyResource,
		resources.NewFloatingIPResource,
		resources.NewSubnetResource,
	}
}

// DataSources (none for now)
func (p *KrutrimProvider) DataSources(
	ctx context.Context,
) []func() datasource.DataSource {

	return []func() datasource.DataSource{}
}

// Provider factory
func NewProvider(version string) func() provider.Provider {

	return func() provider.Provider {

		return &KrutrimProvider{
			version: version,
		}
	}
}