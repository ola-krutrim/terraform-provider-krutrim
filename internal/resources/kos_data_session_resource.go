// package resources

// import (
// 	"context"

// 	"github.com/hashicorp/terraform-plugin-framework/datasource"
// 	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
// 	"github.com/hashicorp/terraform-plugin-framework/types"

// 	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
// )

// var _ datasource.DataSource = &SessionDataSource{}
// var _ datasource.DataSourceWithConfigure = &SessionDataSource{}

// type SessionDataSource struct {
// 	client *krutrim.Client
// }

// func NewSessionDataSource() datasource.DataSource {
// 	return &SessionDataSource{}
// }

// type SessionModel struct {
// 	AccessKey types.String `tfsdk:"access_key"`
// 	SecretKey types.String `tfsdk:"secret_key"`
// 	Region    types.String `tfsdk:"region"`
// 	Tier      types.String `tfsdk:"tier"`

// 	SessionToken types.String `tfsdk:"session_token"`
// }

// func (d *SessionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
// 	resp.TypeName = req.ProviderTypeName + "_kos_session"
// }

// func (d *SessionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
// 	resp.Schema = schema.Schema{
// 		Attributes: map[string]schema.Attribute{
// 			"access_key": schema.StringAttribute{Required: true},
// 			"secret_key": schema.StringAttribute{Required: true, Sensitive: true},
// 			"region":     schema.StringAttribute{Required: true},
// 			"tier":       schema.StringAttribute{Required: true},
// 			"session_token": schema.StringAttribute{
// 				Computed:  true,
// 			},
// 		},
// 	}
// }

// func (d *SessionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
// 	if req.ProviderData == nil {
// 		return
// 	}

// 	client, ok := req.ProviderData.(*krutrim.Client)
// 	if !ok {
// 		resp.Diagnostics.AddError(
// 			"Unexpected Provider Data Type",
// 			"Expected *krutrim.Client",
// 		)
// 		return
// 	}

// 	d.client = client
// }

// func (d *SessionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

// 	// ✅ client safety
// 	if d.client == nil {
// 		resp.Diagnostics.AddError("Client not initialized", "Provider client is nil")
// 		return
// 	}

// 	var state SessionModel

// 	// ✅ read config
// 	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	// ✅ API call
// 	res, err := d.client.Ko.V1.Sessions.Activate(
// 		ctx,
// 		state.Region.ValueString(),
// 		state.Tier.ValueString(),
// 		krutrim.ActivateSessionParams{
// 			AccessKey: state.AccessKey.ValueString(),
// 			SecretKey: state.SecretKey.ValueString(),
// 		},
// 	)

// 	if err != nil {
// 		resp.Diagnostics.AddError("Session failed", err.Error())
// 		return
// 	}

// 	// ✅ correct key
// 	token := res["session_token"]
// 	if token == "" {
// 		resp.Diagnostics.AddError("Missing field", "session_token not found")
// 		return
// 	}

// 	state.SessionToken = types.StringValue(token)

// 	// ✅ set state safely
// 	diags := resp.State.Set(ctx, &state)
// 	resp.Diagnostics.Append(diags...)
// }



package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var _ resource.Resource = &SessionResource{}
var _ resource.ResourceWithConfigure = &SessionResource{}

type SessionResource struct {
	client *krutrim.Client
}

func NewSessionResource() resource.Resource {
	return &SessionResource{}
}

type SessionModel struct {
	ID types.String `tfsdk:"id"`

	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
	Region    types.String `tfsdk:"region"`
	Tier      types.String `tfsdk:"tier"`

	SessionToken types.String `tfsdk:"session_token"`
}

/* ================= METADATA ================= */

func (r *SessionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kos_session"
}

/* ================= SCHEMA ================= */

func (r *SessionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"id": schema.StringAttribute{
				Computed: true,
			},

			"access_key": schema.StringAttribute{
				Required: true,
			},

			"secret_key": schema.StringAttribute{
				Required:  true,
			},

			"region": schema.StringAttribute{
				Required: true,
			},

			"tier": schema.StringAttribute{
				Required: true,
			},

			"session_token": schema.StringAttribute{
				Computed:  true,
			},
		},
	}
}

/* ================= CONFIGURE ================= */

func (r *SessionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			"Expected *krutrim.Client",
		)
		return
	}

	r.client = client
}

/* ================= CREATE ================= */

func (r *SessionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	// ✅ client safety
	if r.client == nil {
		resp.Diagnostics.AddError("Client not initialized", "Provider client is nil")
		return
	}

	var plan SessionModel

	// read plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// API call
	res, err := r.client.Ko.V1.Sessions.Activate(
		ctx,
		plan.Region.ValueString(),
		plan.Tier.ValueString(),
		krutrim.ActivateSessionParams{
			AccessKey: plan.AccessKey.ValueString(),
			SecretKey: plan.SecretKey.ValueString(),
		},
	)

	if err != nil {
		resp.Diagnostics.AddError("Session creation failed", err.Error())
		return
	}

	// extract token
	token := res["session_token"]
	if token == "" {
		resp.Diagnostics.AddError("Missing field", "session_token not found")
		return
	}

	plan.SessionToken = types.StringValue(token)

	// ✅ use token as ID (important)
	plan.ID = types.StringValue(token)

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

/* ================= READ ================= */

func (r *SessionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// ⚠️ No API to read session → keep state as-is
}

/* ================= DELETE ================= */

func (r *SessionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// ⚠️ No delete API → just remove from state
	resp.State.RemoveResource(ctx)
}

/* ================= UPDATE ================= */

func (r *SessionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// ❌ session cannot be updated → force recreate
	resp.Diagnostics.AddError("Update not supported", "Session must be recreated")
}


