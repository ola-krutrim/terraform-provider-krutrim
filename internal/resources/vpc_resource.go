package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	krutrim "github.com/ola-silicon/krutrim-go-sdk"
	"github.com/ola-silicon/krutrim-go-sdk/option"
)

const (
	// Polling intervals
	createPollInterval = 10 * time.Second
	deletePollInterval = 10 * time.Second
	
	// Timeouts
	createTimeout = 10 * time.Minute
	deleteTimeout = 10 * time.Minute
	
	// Task statuses
	taskStatusSuccess   = "success"
	taskStatusFailed    = "failed"
	taskStatusPending   = "pending"
	taskStatusProcessing = "processing"
	
	// Field types
	fieldTypeVPC    = "vpc"
	fieldTypeSubnet = "subnet"
	fieldTypeNetwork = "network"
	
	// IP versions
	ipVersionIPv4 = "4"
	ipVersionIPv6 = "6"
)

/*
=================================================
Resource Implementation
=================================================
*/

var (
	_ resource.Resource                = &VPCResource{}
	_ resource.ResourceWithConfigure   = &VPCResource{}
	_ resource.ResourceWithImportState = &VPCResource{}
)

type VPCResource struct {
	client *krutrim.Client
}

/*
=================================================
State Model
=================================================
*/

type VPCModel struct {
	// Computed attributes
	ID       types.String `tfsdk:"id"`
	SubnetID types.String `tfsdk:"subnet_id"`
	NetworkID types.String `tfsdk:"network_id"`

	
	// Required attributes
	Region      types.String `tfsdk:"region"`
	Name        types.String `tfsdk:"name"`
	NetworkName types.String `tfsdk:"network_name"`
	SubnetName  types.String `tfsdk:"subnet_name"`
	CIDR        types.String `tfsdk:"cidr"`
	GatewayIP   types.String `tfsdk:"gateway_ip"`
	
	// Optional attributes
	Description  types.String `tfsdk:"description"`
	SubnetDescription types.String `tfsdk:"subnet_description"`

	Enabled      types.Bool   `tfsdk:"enabled"`
	AdminStateUp types.Bool   `tfsdk:"admin_state_up"`
	IPVersion    types.String `tfsdk:"ip_version"`
	Ingress	  types.Bool   `tfsdk:"ingress"`
	Egress	  types.Bool   `tfsdk:"egress"`
}

/*
=================================================
API Response Models
=================================================
*/

type TaskStatusResponse struct {
	Data struct {
		Tasks []struct {
			Field  string `json:"field"`
			Status string `json:"status"`
			KRN    string `json:"krn"`
			Error  string `json:"error,omitempty"`

			SubnetList []struct {
				Status string `json:"status"`
				KRN    string `json:"krn"`
				Error  string `json:"error,omitempty"`
			} `json:"subnet_list,omitempty"`
		} `json:"tasks"`
	} `json:"data"`
	Error struct {
		Message string `json:"message,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

type CreateVPCResponse struct {
	TaskID string `json:"task_id"`
	Error  struct {
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
}


type SearchVPCResponse struct {
	Data struct {
		VPCs []struct {
			ID     string `json:"id"`
			KRN    string `json:"krn"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"vpcs"`
	} `json:"data"`
	Error struct {
		Message string `json:"message,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

/*
=================================================
Constructor
=================================================
*/

func NewVPCResource() resource.Resource {
	return &VPCResource{}
}

/*
=================================================
Metadata
=================================================
*/

func (r *VPCResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

/*
=================================================
Schema
=================================================
*/

func (r *VPCResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VPC (Virtual Private Cloud) resource with associated network and subnet.",
		
		Attributes: map[string]schema.Attribute{
			// Computed
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier (KRN) of the VPC",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			
			"subnet_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier (KRN) of the subnet",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			
			// Required
			"region": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The region where the VPC will be created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the VPC",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			
			"network_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the network within the VPC",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			
			"subnet_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the subnet",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			
			"cidr": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The CIDR block for the subnet (e.g., 10.0.0.0/24)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			
			"gateway_ip": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The gateway IP address for the subnet",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			
			// Optional
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Description of the VPC",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the VPC is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			
			"admin_state_up": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Administrative state of the network",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			
			"ip_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(ipVersionIPv4),
				MarkdownDescription: "IP version (4 for IPv4, 6 for IPv6)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				MarkdownDescription: "Description of the subnet",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier (KRN) of the network",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ingress": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"egress": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},


		},
	}
}

/*
=================================================
Configure
=================================================
*/

func (r *VPCResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *krutrim.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

/*
=================================================
CREATE
=================================================
*/

func (r *VPCResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan VPCModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating VPC", map[string]interface{}{
		"name":   plan.Name.ValueString(),
		"region": plan.Region.ValueString(),
	})

	// Validate IP version
	if err := r.validateIPVersion(plan.IPVersion.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid IP Version", err.Error())
		return
	}

	// Build creation parameters
	params := krutrim.CreateVpcAsyncNewParams{
		XRegion: plan.Region.ValueString(),
		Vpc: krutrim.Vpc{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Enabled:     plan.Enabled.ValueBool(),
		},
		Network: krutrim.Network{
			Name:         plan.NetworkName.ValueString(),
			AdminStateUp: plan.AdminStateUp.ValueBool(),
		},
		
	}
	if !plan.SubnetName.IsNull() {
		params.Subnet = krutrim.Subnet{
			Name:        plan.SubnetName.ValueString(),
			Description: plan.SubnetDescription.ValueString(),
			CIDR:        plan.CIDR.ValueString(),
			GatewayIP:   plan.GatewayIP.ValueString(),
			IPVersion:   plan.IPVersion.ValueString(),
			Ingress:     plan.Ingress.ValueBool(), 
			Egress:      plan.Egress.ValueBool(),  
		}
	}

	// Call API to create VPC

	var httpResp *http.Response

	err := r.client.CreateVpcAsync.New(
		ctx,
		params,
		option.WithResponseInto(&httpResp),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"VPC Creation Failed",
			err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Response Read Failed",
			fmt.Sprintf("Unable to read API response: %s", err.Error()),
		)
		return
	}
	
	if httpResp.StatusCode >= 400 {
		resp.Diagnostics.AddError(
			"VPC Creation API Error",
			fmt.Sprintf(
				"Status: %d, Body: %s",
				httpResp.StatusCode,
				string(body),
			),
		)
		return
	}

	var createResp CreateVPCResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Response Parse Failed",
			fmt.Sprintf("Unable to parse API response: %s\nBody: %s", err.Error(), string(body)),
		)
		return
	}

	if createResp.Error.Message != "" {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("API returned error: %s", createResp.Error.Message),
		)
		return
	}

	if createResp.TaskID == "" {
		resp.Diagnostics.AddError(
			"Invalid API Response",
			fmt.Sprintf("No task_id received in response. Body: %s", string(body)),
		)
		return
	}

	tflog.Info(ctx, "VPC creation initiated", map[string]interface{}{
		"task_id": createResp.TaskID,
	})

	// Wait for VPC to be provisioned
	vpcID, subnetID, networkID, err := r.waitForVPCCreation(
		ctx,
		createResp.TaskID,
		plan.Region.ValueString(),
	)

	

	if err != nil {
		resp.Diagnostics.AddError(
			"VPC Provision Failed",
			fmt.Sprintf("VPC creation timed out or failed: %s", err.Error()),
		)
		return
	}
	if subnetID == "" {
		// Terraform warning (shown to user)
		resp.Diagnostics.AddWarning(
			"VPC subnet creation skipped",
			"The subnet requested during VPC creation was not provisioned by the VPC API. "+
				"If you are managing subnets using krutrim_subnet, this warning can be safely ignored.",
		)
	
		// Provider log (for debugging)
		tflog.Warn(ctx, "Subnet was not created during VPC creation", map[string]interface{}{
			"vpc_id": vpcID,
		})
	}

	// Update state with created resource IDs
	plan.ID = types.StringValue(vpcID)
	if subnetID != "" {
		plan.SubnetID = types.StringValue(subnetID)
	} else {
		plan.SubnetID = types.StringNull()
	}
	if networkID != "" {
		plan.NetworkID = types.StringValue(networkID)
	} else {
		plan.NetworkID = types.StringNull()
	}

	tflog.Info(ctx, "VPC created successfully", map[string]interface{}{
		"vpc_id":    vpcID,
		"subnet_id": subnetID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

/*
=================================================
READ
=================================================
*/

func (r *VPCResource) Read(
    ctx context.Context,
    req resource.ReadRequest,
    resp *resource.ReadResponse,
) {
    var state VPCModel

    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    vpcID := state.ID.ValueString()
    region := state.Region.ValueString()

    var httpResp *http.Response

    err := r.client.DescribeVpc.Get(
        ctx,
        krutrim.DescribeVpcGetParams{
            VpcID:   vpcID,
            VpcName: "", // REQUIRED to avoid SDK validation
            XRegion: region,
        },
        option.WithResponseInto(&httpResp),
    )

    if err != nil {
        // IMPORTANT: On ANY error, KEEP STATE
        tflog.Warn(ctx, "DescribeVpc failed, keeping resource in state", map[string]interface{}{
            "vpc_id": vpcID,
            "error":  err.Error(),
        })
        return
    }

    defer httpResp.Body.Close()

    switch httpResp.StatusCode {

    case http.StatusOK:
        // VPC exists — do nothing
        tflog.Debug(ctx, "VPC exists", map[string]interface{}{
            "vpc_id": vpcID,
        })
        resp.State.Set(ctx, &state)
        return

    case http.StatusNotFound:
        // VPC is really gone — NOW remove from state
        tflog.Info(ctx, "VPC not found, removing from state", map[string]interface{}{
            "vpc_id": vpcID,
        })
        resp.State.RemoveResource(ctx)
        return

    default:
        // Unknown error — KEEP STATE
        tflog.Warn(ctx, "Unexpected DescribeVpc response, keeping state", map[string]interface{}{
            "vpc_id": vpcID,
            "status": httpResp.StatusCode,
        })
        return
    }
}

/*
=================================================
UPDATE
=================================================
*/

func (r *VPCResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan VPCModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}


	tflog.Warn(ctx, "Update called on VPC resource - this should trigger replacement")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

/*
=================================================
DELETE
=================================================
*/

func (r *VPCResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state VPCModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := state.Region.ValueString()
	vpcID := state.ID.ValueString()
	subnetID := state.SubnetID.ValueString()

	tflog.Debug(ctx, "Deleting VPC", map[string]interface{}{
		"vpc_id":    vpcID,
		"subnet_id": subnetID,
		"region":    region,
	})

	// Delete subnet first (if exists)
	if subnetID != "" {
		tflog.Debug(ctx, "Deleting subnet", map[string]interface{}{
			"subnet_id": subnetID,
		})

		err := r.client.DeleteSubnet.Delete(
			ctx,
			krutrim.DeleteSubnetDeleteParams{
				SubnetID: subnetID,
				VpcID:    vpcID,
				XRegion:  region,
			},
		)

		if err != nil {
			tflog.Warn(ctx, "Subnet deletion failed, continuing with VPC deletion", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Wait for subnet deletion to complete
		time.Sleep(20 * time.Second)
	}

	// Delete VPC
	err := r.client.DeleteVpc.Delete(
		ctx,
		krutrim.DeleteVpcDeleteParams{
			VpcID:   vpcID,
			XRegion: region,
		},
	)

	if err != nil {

		if strings.Contains(err.Error(), "409") ||
		   strings.Contains(strings.ToLower(err.Error()), "cannot delete vpc") {
	
			resp.Diagnostics.AddWarning(
				"VPC deletion blocked by dependent resources",
				"The VPC still has associated resources (most commonly volumes that are still deleting). "+
					"Please wait for all dependent resources to be fully removed and run `terraform destroy` again.",
			)
		}
	
		resp.Diagnostics.AddError(
			"VPC Deletion Failed",
			fmt.Sprintf("Unable to delete VPC %s: %s", vpcID, err.Error()),
		)
		return
	}

	tflog.Info(ctx, "VPC deletion API call successful, waiting for confirmation", map[string]interface{}{
		"vpc_id": vpcID,
	})

	// Wait for deletion to complete
	if err := r.waitForVPCDeletion(ctx, vpcID, region); err != nil {
		resp.Diagnostics.AddError(
			"VPC Deletion Timeout",
			fmt.Sprintf("VPC deletion did not complete within timeout: %s", err.Error()),
		)
		return
	}

	tflog.Info(ctx, "VPC deleted successfully", map[string]interface{}{
		"vpc_id": vpcID,
	})
}

/*
=================================================
Import State
=================================================
*/

func (r *VPCResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import format: <region>:<vpc_id>:<subnet_id>
	parts := strings.Split(req.ID, ":")
	
	if len(parts) < 2 || len(parts) > 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: <region>:<vpc_id> or <region>:<vpc_id>:<subnet_id>",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	
	if len(parts) == 3 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subnet_id"), parts[2])...)
	}
}

/*
=================================================
Helper Functions
=================================================
*/

// waitForVPCCreation polls the task status until VPC is created or timeout
func (r *VPCResource) waitForVPCCreation(
	ctx context.Context,
	taskID string,
	region string,
) (vpcID string, subnetID string, networkID string, err error) {

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	ticker := time.NewTicker(createPollInterval)
	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():
			return "", "", "", fmt.Errorf("timeout after %v waiting for VPC creation: %w",
				createTimeout, ctx.Err())

		case <-ticker.C:
			vpcID, subnetID, networkID, status, err := r.checkTaskStatus(ctx, taskID, region)
			if err != nil {
				tflog.Warn(ctx, "Task status check failed, retrying", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			switch status {

			case taskStatusSuccess:
				if vpcID == "" {
					return "", "", "", fmt.Errorf("task succeeded but no VPC ID returned")
				}
				return vpcID, subnetID, networkID, nil

			case taskStatusFailed:
				return "", "", "", fmt.Errorf("VPC creation failed")

			case taskStatusPending, taskStatusProcessing:
				tflog.Debug(ctx, "VPC creation in progress", map[string]interface{}{
					"status": status,
				})
				continue

			default:
				tflog.Warn(ctx, "Unknown task status received", map[string]interface{}{
					"status": status,
				})
				continue
			}
		}
	}
}

func (r *VPCResource) checkTaskStatus(
	ctx context.Context,
	taskID string,
	region string,
) (vpcID string, subnetID string, networkID string, status string, err error) {

	var httpResp *http.Response

	err = r.client.GetVpcTaskStatus.New(
		ctx,
		krutrim.GetVpcTaskStatusNewParams{
			TaskID:  taskID,
			XRegion: region,
		},
		option.WithResponseInto(&httpResp),
	)

	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get task status: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		body, _ := io.ReadAll(httpResp.Body)
		return "", "", "", "", fmt.Errorf(
			"task status API error: status=%d body=%s",
			httpResp.StatusCode,
			string(body),
		)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to read response: %w", err)
	}

	var data TaskStatusResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	var vpcStatus string

	for _, task := range data.Data.Tasks {

		if task.Field == fieldTypeVPC {
			vpcID = task.KRN
			vpcStatus = task.Status

			if task.Status == taskStatusFailed {
				return "", "", "", taskStatusFailed,
					fmt.Errorf("VPC task failed")
			}
		}

		if task.Field == fieldTypeNetwork {
			networkID = task.KRN
		}

		if task.Field == fieldTypeSubnet {
			for _, subnet := range task.SubnetList {

				if subnet.KRN != "" {
					subnetID = subnet.KRN
				}

				if subnet.Error != "" {
					tflog.Warn(ctx, "Subnet creation failed", map[string]interface{}{
						"error": subnet.Error,
					})
				}
			}
		}
	}

	if vpcStatus == "" {
		return "", "", "", taskStatusFailed,
			fmt.Errorf("no VPC task found in task status response")
	}

	return vpcID, subnetID, networkID, vpcStatus, nil
}

// waitForVPCDeletion polls until VPC is deleted or timeout
func (r *VPCResource) waitForVPCDeletion(
	ctx context.Context,
	vpcID string,
	region string,
) error {
	ticker := time.NewTicker(deletePollInterval)
	defer ticker.Stop()

	timeout := time.After(deleteTimeout)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())

		case <-timeout:
			return fmt.Errorf("timeout after %v waiting for VPC deletion", deleteTimeout)

		case <-ticker.C:
			exists, err := r.vpcExists(ctx, vpcID, region)
			
			if err != nil {
				tflog.Warn(ctx, "VPC existence check failed, retrying", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			if !exists {
				tflog.Info(ctx, "VPC confirmed deleted", map[string]interface{}{
					"vpc_id": vpcID,
				})
				return nil // VPC successfully deleted
			}

			tflog.Debug(ctx, "VPC still exists, waiting for deletion", map[string]interface{}{
				"vpc_id": vpcID,
			})
		}
	}
}


func (r *VPCResource) vpcExists(
	ctx context.Context,
	vpcID string,
	region string,
) (bool, error) {
	var httpResp *http.Response

	// Call SearchVpc API with response capture
	err := r.client.SearchVpc.List(
		ctx,
		krutrim.SearchVpcListParams{
			XRegion: region,
			Page:    1,
			Size:    100, // Increased to ensure we get all VPCs
		},
		option.WithResponseInto(&httpResp),
	)

	// If API call fails, check if it's a 404 or "not found" error
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "404") || 
		   strings.Contains(strings.ToLower(errStr), "not found") ||
		   strings.Contains(strings.ToLower(errStr), "no vpc") {
			tflog.Debug(ctx, "VPC list returned 404 or not found, VPC deleted", map[string]interface{}{
				"vpc_id": vpcID,
			})
			return false, nil
		}
		
		// For other errors, log and assume VPC might still exist
		tflog.Warn(ctx, "Failed to check VPC existence", map[string]interface{}{
			"error": err.Error(),
		})
		return true, fmt.Errorf("failed to list VPCs: %w", err)
	}

	// If we have a response, parse it
	if httpResp != nil {
		defer httpResp.Body.Close()
		
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			tflog.Warn(ctx, "Failed to read VPC list response", map[string]interface{}{
				"error": err.Error(),
			})
			return true, fmt.Errorf("failed to read response: %w", err)
		}

		tflog.Trace(ctx, "VPC list response", map[string]interface{}{
			"body": string(body),
		})

		// Try to parse the response
		var searchResp SearchVPCResponse
		if err := json.Unmarshal(body, &searchResp); err != nil {
			// If parsing fails, log the body and continue
			tflog.Warn(ctx, "Failed to parse VPC list response", map[string]interface{}{
				"error": err.Error(),
				"body":  string(body),
			})
			// Don't fail here - we'll use fallback logic below
		} else {
			// Successfully parsed - check if VPC is in the list
			for _, vpc := range searchResp.Data.VPCs {
				// Check both ID and KRN fields
				if vpc.ID == vpcID || vpc.KRN == vpcID {
					tflog.Debug(ctx, "VPC found in list", map[string]interface{}{
						"vpc_id": vpcID,
						"status": vpc.Status,
					})
					return true, nil
				}
			}
			
			// VPC not found in list - it's been deleted
			tflog.Debug(ctx, "VPC not found in list, confirmed deleted", map[string]interface{}{
				"vpc_id":     vpcID,
				"total_vpcs": len(searchResp.Data.VPCs),
			})
			return false, nil
		}
	}

	// Fallback: If we got here, we couldn't determine status
	// Try a direct GET call if available
	tflog.Debug(ctx, "Unable to parse list response, assuming VPC still exists", map[string]interface{}{
		"vpc_id": vpcID,
	})
	return true, nil
}

// validateIPVersion validates the IP version value
func (r *VPCResource) validateIPVersion(version string) error {
	if version == "" {
		return nil // Will use default
	}

	if version != ipVersionIPv4 && version != ipVersionIPv6 {
		return fmt.Errorf("ip_version must be '4' (IPv4) or '6' (IPv6), got: %s", version)
	}

	return nil
}