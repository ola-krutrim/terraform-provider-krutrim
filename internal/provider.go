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

// ==============================
// Provider Config Model
// ==============================
type KrutrimProviderModel struct {
	BaseURL types.String `tfsdk:"base_url"`

	Email      types.String `tfsdk:"email"`
	Password   types.String `tfsdk:"password"`
	AccountID  types.String `tfsdk:"account_id"`
	IsRootUser types.Bool   `tfsdk:"is_root_user"`

	// 🔥 NEW (Programmatic Auth)
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

// ==============================
// Metadata
// ==============================
func (p *KrutrimProvider) Metadata(
	ctx context.Context,
	req provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "krutrim"
	resp.Version = p.version
}

// ==============================
// Schema
// ==============================
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

			// 🔹 Email login
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

			// 🔥 Programmatic login
			"access_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"secret_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// ==============================
// Configure
// ==============================
func (p *KrutrimProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var config KrutrimProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ==========================================
	// 🔥 PROGRAMMATIC AUTH (Priority)
	// ==========================================
	if !config.AccessKey.IsNull() && !config.SecretKey.IsNull() {

		if config.BaseURL.IsNull() {
			resp.Diagnostics.AddError("Missing base_url", "Required for programmatic authentication")
			return
		}

		authResp, err := auth.SignInProgrammatic(auth.ProgrammaticAuthConfig{
			BaseURL:   config.BaseURL.ValueString(),
			AccountID: config.AccountID.ValueString(),
			AccessKey: config.AccessKey.ValueString(),
			SecretKey: config.SecretKey.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Programmatic Authentication Failed", err.Error())
			return
		}

		opts := []option.RequestOption{
			option.WithBaseURL(config.BaseURL.ValueString()),
			option.WithHeader("Authorization", "Bearer "+authResp.Token),
		}

		client := krutrim.NewClient(opts...)
		resp.ResourceData = &client
		resp.DataSourceData = &client
		return
	}

	// ==========================================
	// 🔥 EMAIL / PASSWORD AUTH
	// ==========================================
	if config.Email.IsNull() || config.Password.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Credentials",
			"Provide either email/password OR access_key/secret_key",
		)
		return
	}

	if config.BaseURL.IsNull() {
		resp.Diagnostics.AddError(
			"Missing base_url",
			"base_url is required",
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
		resp.Diagnostics.AddError("Authentication Failed", err.Error())
		return
	}

	opts := []option.RequestOption{
		option.WithBaseURL(config.BaseURL.ValueString()),
		option.WithHeader("Authorization", "Bearer "+authResp.AccessToken),
	}

	client := krutrim.NewClient(opts...)
	resp.ResourceData = &client
	resp.DataSourceData = &client
}

// ==============================
// Resources
// ==============================
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
		resources.NewKOSBucketResource,
		resources.NewKOSObjectResource,
		resources.NewAccessKeyResource,
		resources.NewDNSZoneResource,
		resources.NewDNSRecordResource,
		resources.NewLoadBalancerResource,
		resources.NewTargetGroupResource,
		resources.NewKcmCertResource,
		resources.NewSessionResource,

		// IAM
		resources.NewIAMUserResource,
		resources.NewIAMRoleResource,
		resources.NewIAMUserRoleBindingResource,
		resources.NewIAMProgrammaticAccessResource,
	}
}

// ==============================
// DataSources
// ==============================
func (p *KrutrimProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		resources.NewIAMPoliciesDataSource,
		resources.NewIAMRolesDataSource,
	}
}

// ==============================
// Provider Factory
// ==============================
func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KrutrimProvider{
			version: version,
		}
	}
}