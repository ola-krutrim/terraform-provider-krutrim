package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	krutrim "github.com/ola-silicon/krutrim-go-sdk"
	"github.com/ola-silicon/krutrim-go-sdk/option"
)

var _ resource.Resource = &SSHKeyResource{}
var _ resource.ResourceWithConfigure = &SSHKeyResource{}

// =======================
// Models
// =======================

type SSHKeyModel struct {
	ID        types.String `tfsdk:"id"`
	KeyName   types.String `tfsdk:"key_name"`
	PublicKey types.String `tfsdk:"public_key"`
	Region    types.String `tfsdk:"region"`
}

// Response structures matching actual API
type SSHKeyCreateResponse struct {
	ID        string `json:"_id"`
	UUID      string `json:"uuid"`
	KeyName   string `json:"keyName"`
	PublicKey string `json:"publicKey"`
	Region    string `json:"region"`
}

type SSHKeySearchResponse struct {
	SSHKeys []struct {
		ID        string `json:"_id"`
		UUID      string `json:"uuid"`
		KeyName   string `json:"keyName"`
		PublicKey string `json:"publicKey"`
		Region    string `json:"region"`
	} `json:"sshKeys"` // ← Fixed: was "data", should be "sshKeys"
}

// =======================
// Resource
// =======================

type SSHKeyResource struct {
	client *krutrim.Client
}

func NewSSHKeyResource() resource.Resource {
	return &SSHKeyResource{}
}

func (r *SSHKeyResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_sshkey"
}

// =======================
// Schema
// =======================

func (r *SSHKeyResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Manages an SSH key in Krutrim Cloud",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SSH key ID (UUID)",
			},
			"key_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the SSH key",
			},
			"public_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "SSH public key content",
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "Region where the key is stored (e.g., In-Bangalore-1)",
			},
		},
	}
}

// =======================
// Configure
// =======================

func (r *SSHKeyResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	_ *resource.ConfigureResponse,
) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*krutrim.Client)
	}
}

// =======================
// CREATE
// =======================

func (r *SSHKeyResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SSHKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create params
	params := krutrim.SshkeyNewParams{
		KeyName:   plan.KeyName.ValueString(),
		PublicKey: plan.PublicKey.ValueString(),
		XRegion:   plan.Region.ValueString(),
	}

	// Call API and capture HTTP response
	var httpResp *http.Response
	err := r.client.Sshkey.New(
		ctx,
		params,
		option.WithResponseInto(&httpResp),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Key creation failed",
			"Could not create SSH key: "+err.Error(),
		)
		return
	}

	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)

	// Parse create response
	var createResult SSHKeyCreateResponse
	if err := json.Unmarshal(body, &createResult); err != nil {
		resp.Diagnostics.AddError(
			"Failed to parse create response",
			"Error: "+err.Error()+"\nResponse: "+string(body),
		)
		return
	}

	// Use UUID as ID (preferred over _id/krn)
	if createResult.UUID != "" {
		plan.ID = types.StringValue(createResult.UUID)
	} else if createResult.ID != "" {
		// Fallback to _id if UUID not present
		plan.ID = types.StringValue(createResult.ID)
	} else {
		resp.Diagnostics.AddError(
			"No ID in create response",
			"API returned success but no ID. Response: "+string(body),
		)
		return
	}

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// =======================
// READ
// =======================

func (r *SSHKeyResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SSHKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build search params
	params := krutrim.SshkeySearchParams{
		KeyName: state.KeyName.ValueString(),
	}

	// Call API
	var httpResp *http.Response
	err := r.client.Sshkey.Search(
		ctx,
		params,
		option.WithResponseInto(&httpResp),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Key read failed",
			"Could not read SSH key: "+err.Error(),
		)
		return
	}

	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)

	// Parse search response (uses "sshKeys" not "data")
	var searchResult SSHKeySearchResponse
	if err := json.Unmarshal(body, &searchResult); err != nil {
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddWarning(
			"Failed to search the key",
			"Removing SSH key from state",
		)
		return
	}

	// Check if key was found
	if len(searchResult.SSHKeys) == 0 {
		// Key doesn't exist anymore - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Find the key with matching ID
	var found bool
	for _, key := range searchResult.SSHKeys {
		// Match by UUID or _id
		if key.UUID == state.ID.ValueString() || key.ID == state.ID.ValueString() {
			// Update state with API data
			state.ID = types.StringValue(key.UUID)
			state.KeyName = types.StringValue(key.KeyName)
			state.PublicKey = types.StringValue(key.PublicKey)
			state.Region = types.StringValue(key.Region)
			found = true
			break
		}
	}

	if !found {
		// Key with this ID doesn't exist - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// =======================
// UPDATE
// =======================

func (r *SSHKeyResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan SSHKeyModel
	var state SSHKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// =======================
// DELETE
// =======================

func (r *SSHKeyResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SSHKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the SSH key (use UUID as ID)
	err := r.client.Sshkey.Delete(
		ctx,
		state.ID.ValueString(),
		option.WithHeader("x-region", state.Region.ValueString()),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Key deletion failed",
			"Could not delete SSH key ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
}
