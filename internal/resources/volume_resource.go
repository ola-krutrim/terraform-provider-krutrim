package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	krutrim "github.com/ola-silicon/krutrim-go-sdk"
	"github.com/ola-silicon/krutrim-go-sdk/packages/param"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"time"
)

var _ resource.Resource = &VolumeResource{}
var _ resource.ResourceWithConfigure = &VolumeResource{}

// =======================
// Model
// =======================

type VolumeModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Size        types.Int64  `tfsdk:"size"`
	VolumeType  types.String `tfsdk:"volume_type"`
	Multiattach types.Bool   `tfsdk:"multiattach"`

	// Source (optional)
	SourceID   types.String `tfsdk:"source_id"`
	SourceType types.String `tfsdk:"source_type"`

	// Metadata (optional)
	MetadataEnvironment types.String `tfsdk:"metadata_environment"`
	MetadataTeam        types.String `tfsdk:"metadata_team"`

	// Headers (required by API)
	KTenantID types.String `tfsdk:"k_tenant_id"`
}

// =======================
// Resource
// =======================

type VolumeResource struct {
	client *krutrim.Client
}

func NewVolumeResource() resource.Resource {
	return &VolumeResource{}
}

func (r *VolumeResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

// =======================
// Schema
// =======================

func (r *VolumeResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Manages a Krutrim Block Storage Volume",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Volume ID",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Volume name",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Volume description",
			},
			"size": schema.Int64Attribute{
				Required:    true,
				Description: "Volume size in GB",
			},
			"volume_type": schema.StringAttribute{
				Required:    true,
				Description: "Volume type (e.g., HNSS)",
			},
			"multiattach": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable multiattach",
			},
			"source_id": schema.StringAttribute{
				Optional:    true,
				Description: "Source volume/snapshot ID",
			},
			"source_type": schema.StringAttribute{
				Optional:    true,
				Description: "Source type (volume or snapshot)",
			},
			"metadata_environment": schema.StringAttribute{
				Optional:    true,
				Description: "Environment metadata tag",
			},
			"metadata_team": schema.StringAttribute{
				Optional:    true,
				Description: "Team metadata tag",
			},
			"k_tenant_id": schema.StringAttribute{
				Required:    true,
				Description: "Tenant ID (k-tenant-id header)",
			},
		},
	}
}

// =======================
// Configure
// =======================

func (r *VolumeResource) Configure(
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

func (r *VolumeResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan VolumeModel

	// Read plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build SDK params
	params := krutrim.KBV1VolumeNewParams{
		Name:       plan.Name.ValueString(),
		Size:       plan.Size.ValueInt64(),
		Volumetype: plan.VolumeType.ValueString(),
		KTenantID:  plan.KTenantID.ValueString(),
	}

	// Optional description
	if !plan.Description.IsNull() {
		params.Description = param.NewOpt(plan.Description.ValueString())
	}

	// Optional multiattach
	if !plan.Multiattach.IsNull() {
		params.Multiattach = param.NewOpt(plan.Multiattach.ValueBool())
	}

	// Optional metadata
	if !plan.MetadataEnvironment.IsNull() || !plan.MetadataTeam.IsNull() {
		params.Metadata = make(map[string]string)
		if !plan.MetadataEnvironment.IsNull() {
			params.Metadata["environment"] = plan.MetadataEnvironment.ValueString()
		}
		if !plan.MetadataTeam.IsNull() {
			params.Metadata["team"] = plan.MetadataTeam.ValueString()
		}
	}

	// Optional source
	if !plan.SourceID.IsNull() || !plan.SourceType.IsNull() {
		params.Source = krutrim.KBV1VolumeNewParamsSource{}

		if !plan.SourceID.IsNull() {
			params.Source.ID = param.NewOpt(plan.SourceID.ValueString())
		}

		if !plan.SourceType.IsNull() {
			params.Source.Type = param.NewOpt(plan.SourceType.ValueString())
		}
	}

	// Call SDK to create volume
	volumePtr, err := r.client.KBV1.Volumes.New(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating volume",
			"Could not create volume: "+err.Error(),
		)
		return
	}

	// Dereference the pointer to get the actual value
	var volume interface{}
	if volumePtr != nil {
		volume = *volumePtr
	}

	// Debug: Print response
	volumeJSON, _ := json.MarshalIndent(volume, "", "  ")
	fmt.Printf("Volume created successfully:\n%s\n", string(volumeJSON))

	// Extract ID from response
	if volumeMap, ok := volume.(map[string]interface{}); ok {
		// Try to find 'id' field in various possible locations
		if id, exists := volumeMap["id"]; exists {
			plan.ID = types.StringValue(fmt.Sprintf("%v", id))
		} else if vol, exists := volumeMap["volume"]; exists {
			if volMap, ok := vol.(map[string]interface{}); ok {
				if id, exists := volMap["id"]; exists {
					plan.ID = types.StringValue(fmt.Sprintf("%v", id))
				}
			}
		} else {
			// Fallback: use name as ID
			plan.ID = types.StringValue(plan.Name.ValueString())
		}
	} else {
		// If response isn't a map, use name as ID
		plan.ID = types.StringValue(plan.Name.ValueString())
	}

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// =======================
// READ
// =======================

func (r *VolumeResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state VolumeModel

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call SDK to get volume details
	params := krutrim.KBV1VolumeGetParams{
		KTenantID: state.KTenantID.ValueString(),
	}

	volumePtr, err := r.client.KBV1.Volumes.Get(
		ctx,
		state.ID.ValueString(),
		params,
	)
	if err != nil {
		// Check if the error is a 404 (volume not found / deleted outside Terraform)
		errString := err.Error()
		if (strings.Contains(errString, "404") || strings.Contains(errString, "404 Not Found")) &&
			(strings.Contains(errString, "not found") ||
				strings.Contains(errString, "Unable to fetch volume") ||
				strings.Contains(errString, "no longer exists")) {
			// Volume was deleted outside Terraform, remove from state
			resp.Diagnostics.AddWarning(
				"Volume Not Found",
				fmt.Sprintf("Volume %s was deleted outside Terraform. Removing from state.", state.ID.ValueString()),
			)
			resp.State.RemoveResource(ctx)
			return
		}

		// For any other error, report it
		resp.Diagnostics.AddError(
			"Error reading volume",
			"Could not read volume ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Dereference the pointer to get the actual value
	var volume interface{}
	if volumePtr != nil {
		volume = *volumePtr
	}

	// Update state from response
	if volumeMap, ok := volume.(map[string]interface{}); ok {
		if name, exists := volumeMap["name"]; exists {
			state.Name = types.StringValue(fmt.Sprintf("%v", name))
		}
		if size, exists := volumeMap["size"]; exists {
			// Handle both int and float64
			switch v := size.(type) {
			case float64:
				state.Size = types.Int64Value(int64(v))
			case int:
				state.Size = types.Int64Value(int64(v))
			case int64:
				state.Size = types.Int64Value(v)
			}
		}
		if desc, exists := volumeMap["description"]; exists && desc != nil {
			state.Description = types.StringValue(fmt.Sprintf("%v", desc))
		}
	}
	// lines := "check for the output"
	// fmt.Print(lines)
	// tflog.Debug(ctx, lines)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// =======================
// UPDATE
// =======================

func (r *VolumeResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan VolumeModel
	var state VolumeModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Note: Your SDK doesn't have an Update method
	// For now, just update state with plan values
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// =======================
// DELETE
// =======================

func (r *VolumeResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state VolumeModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volumeID := state.ID.ValueString()
	tenantID := state.KTenantID.ValueString()

	tflog.Info(ctx, "Initiating volume deletion", map[string]interface{}{
		"volume_id": volumeID,
	})

	params := krutrim.KBV1VolumeDeleteParams{
		KTenantID: tenantID,
	}

	err := r.client.KBV1.Volumes.Delete(
		ctx,
		volumeID,
		params,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting volume",
			"Could not delete volume ID "+volumeID+": "+err.Error(),
		)
		return
	}

	if err := r.waitForVolumeDeletion(ctx, volumeID, tenantID); err != nil {
		resp.Diagnostics.AddError(
			"Volume deletion timeout",
			err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Volume deleted successfully", map[string]interface{}{
		"volume_id": volumeID,
	})
}



func (r *VolumeResource) waitForVolumeDeletion(
	ctx context.Context,
	volumeID string,
	tenantID string,
) error {

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-timeout:
			return fmt.Errorf("timeout waiting for volume %s deletion", volumeID)

		case <-ticker.C:
			exists, err := r.volumeExists(ctx, volumeID, tenantID)
			if err != nil {
				tflog.Warn(ctx, "Volume existence check failed", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			if !exists {
				return nil
			}

			tflog.Info(ctx, "Volume is still deleting", map[string]interface{}{
				"volume_id": volumeID,
			})
		}
	}
}

func (r *VolumeResource) volumeExists(
	ctx context.Context,
	volumeID string,
	tenantID string,
) (bool, error) {

	params := krutrim.KBV1VolumeGetParams{
		KTenantID: tenantID,
	}

	_, err := r.client.KBV1.Volumes.Get(ctx, volumeID, params)
	if err != nil {
		errStr := strings.ToLower(err.Error())

		if strings.Contains(errStr, "404") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "no longer exists") {
			return false, nil
		}

		return true, err
	}

	return true, nil
}