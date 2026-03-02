// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
// ⚠️ Modified for single-environment usage

package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
	"github.com/ola-krutrim/krutrim-go-sdk/option"
	"github.com/ola-krutrim/terraform-provider-krutrim/internal/auth"

	"github.com/ola-krutrim/terraform-provider-krutrim/internal/resources"
)

// Provider implementation
type KrutrimProvider struct {
	version string
}

// Provider config model
type KrutrimProviderModel struct {
	BaseURL    types.String `tfsdk:"base_url"`

	Email      types.String `tfsdk:"email"`
	Password   types.String `tfsdk:"password"`
	AccountID  types.String `tfsdk:"account_id"`
	IsRootUser types.Bool   `tfsdk:"is_root_user"`
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
			"email": schema.StringAttribute{
				Optional: true,
			},

			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},

			"account_id": schema.StringAttribute{
				Optional: true,
			},

			"is_root_user": schema.BoolAttribute{
				Optional: true,
			},


		},
	}
}

func (p *KrutrimProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var config KrutrimProviderModel

	// Read provider configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}


	if config.Email.IsNull() || config.Password.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Credentials",
			"email and password are required for authentication",
		)
		return
	}

	if config.BaseURL.IsNull() || config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing base_url",
			"base_url is required when using email/password authentication",
		)
		return
	}


	authResp, err := auth.SignIn(auth.AuthConfig{
		BaseURL:    config.BaseURL.ValueString(),
		Email:      config.Email.ValueString(),
		Password:   config.Password.ValueString(),
		AccountID:  config.AccountID.ValueString(),
		IsRootUser: !config.IsRootUser.IsNull() && config.IsRootUser.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Authentication Failed",
			err.Error(),
		)
		return
	}

	opts := []option.RequestOption{
		option.WithEnvironmentProduction(),
		option.WithBaseURL(config.BaseURL.ValueString()),
		option.WithHeader("Authorization", "Bearer "+authResp.AccessToken),
	}
	

	client := krutrim.NewClient(opts...)

	clientPtr := &client
	resp.ResourceData = clientPtr
	resp.DataSourceData = clientPtr
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
		resources.NewInstanceResource,
		resources.NewSecurityGroupResource,
		resources.NewSecurityGroupRuleResource,
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
