package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-silicon/krutrim-go-sdk"
)

var (
	_ resource.Resource              = &InstanceResource{}
	_ resource.ResourceWithConfigure = &InstanceResource{}
)

type InstanceResource struct {
	client *krutrim.Client
}

/*
=================================================
State Model
=================================================
*/

type InstanceModel struct {
	// Computed
	ID types.String `tfsdk:"id"`

	// Required
	Region       types.String `tfsdk:"region"`
	Name         types.String `tfsdk:"name"`
	InstanceType types.String `tfsdk:"instance_type"`
	VpcID        types.String `tfsdk:"vpc_id"`
	SubnetID     types.String `tfsdk:"subnet_id"`
	NetworkID    types.String `tfsdk:"network_id"`
	VolumeType   types.String `tfsdk:"volume_type"`
	VolumeName   types.String `tfsdk:"volume_name"`

	// Optional
	ImageKrn            types.String `tfsdk:"image_krn"`
	SshKeyName          types.String `tfsdk:"sshkey_name"`
	VolumeSize          types.Int64  `tfsdk:"volume_size"`
	IsGPU               types.Bool   `tfsdk:"is_gpu"`
	FloatingIP          types.Bool   `tfsdk:"floating_ip"`
	DeleteOnTermination types.Bool   `tfsdk:"delete_on_termination"`

	// UI-required fields
	SecurityGroups types.List   `tfsdk:"security_groups"`
	UserData       types.String `tfsdk:"user_data"`
	// Computed outputs from API
	FloatingIPAddress types.String `tfsdk:"floating_ip_address"`
	FloatingIPKrn     types.String `tfsdk:"floating_ip_krn"`
	PortKrn           types.String `tfsdk:"port_krn"`
	VolumeKrn         types.String `tfsdk:"volume_krn"`
	VMName           types.String `tfsdk:"vm_name"`
	Status           types.String `tfsdk:"status"`
	PrivateIPAddress types.String `tfsdk:"private_ip_address"`
}

/*
=================================================
Constructor
=================================================
*/

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

/*
=================================================
Metadata
=================================================
*/

func (r *InstanceResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

/*
=================================================
Schema
=================================================
*/

func (r *InstanceResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VM Instance",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"region": schema.StringAttribute{
				Required: true,
			},

			"name": schema.StringAttribute{
				Required: true,
			},

			"instance_type": schema.StringAttribute{
				Required: true,
			},

			"vpc_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"subnet_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"network_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"volume_type": schema.StringAttribute{
				Required: true,
			},

			"volume_name": schema.StringAttribute{
				Required: true,
			},

			"image_krn": schema.StringAttribute{
				Optional: true,
			},

			"sshkey_name": schema.StringAttribute{
				Optional: true,
			},

			"volume_size": schema.Int64Attribute{
				Optional: true,
			},

			"is_gpu": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},

			"floating_ip": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},

			"delete_on_termination": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},

			"security_groups": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Security group KRNs",
			},

			"user_data": schema.StringAttribute{
				Optional: true,
				Sensitive: true,
				MarkdownDescription: "Base64 encoded user data",
			},
			"floating_ip_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"floating_ip_krn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"port_krn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"volume_krn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"vm_name": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"private_ip_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

func (r *InstanceResource) Configure(
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
			fmt.Sprintf("Expected *krutrim.Client, got: %T", req.ProviderData),
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

func (r *InstanceResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan InstanceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}


	if plan.VpcID.IsNull() || plan.VpcID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing vpc_id",
			"vpc_id is required to create an instance, but was not available.",
		)
		return
	}

	if plan.SubnetID.IsNull() || plan.SubnetID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing subnet_id",
			"subnet_id is required to create an instance, but was not available.",
		)
		return
	}

	if plan.NetworkID.IsNull() || plan.NetworkID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing network_id",
			"network_id is required to create an instance, but was not available.",
		)
		return
	}
	plan.FloatingIPKrn = types.StringNull()
	plan.PortKrn       = types.StringNull()
	plan.VolumeKrn     = types.StringNull()
	plan.FloatingIPAddress = types.StringNull()
	plan.PrivateIPAddress  = types.StringNull()
	plan.VMName = types.StringNull()
	plan.Status = types.StringNull()
	// ----------------------------
	// Build API Params (MATCH SDK)
	// ----------------------------

	params := krutrim.HighlvlvpcNewInstanceParams{
		InstanceName: plan.Name.ValueString(),
		InstanceType: plan.InstanceType.ValueString(),
		NetworkID:    plan.NetworkID.ValueString(),
		Region:       plan.Region.ValueString(),
		SubnetID:     plan.SubnetID.ValueString(),
		VpcID:        plan.VpcID.ValueString(),
		Volumetype:   plan.VolumeType.ValueString(),
		VolumeName:   plan.VolumeName.ValueString(),
	}

	// Optional string fields
	if !plan.ImageKrn.IsNull() {
		v := plan.ImageKrn.ValueString()
		params.ImageKrn = &v
	}

	if !plan.SshKeyName.IsNull() {
		v := plan.SshKeyName.ValueString()
		params.SshkeyName = &v
	}

	if !plan.UserData.IsNull() && !plan.UserData.IsUnknown() {
		v := plan.UserData.ValueString()
		params.UserData = &v
	}

	// Optional int
	if !plan.VolumeSize.IsNull() {
		v := plan.VolumeSize.ValueInt64()
		params.VolumeSize = &v
	}

	

	// Optional bools
	isGPU := plan.IsGPU.ValueBool()
	params.IsGPU = &isGPU

	fip := plan.FloatingIP.ValueBool()
	params.FloatingIP = &fip

	del := plan.DeleteOnTermination.ValueBool()
	params.DeleteOnTermination = &del

	// Optional list
	if !plan.SecurityGroups.IsNull() {
		var sgs []string
		plan.SecurityGroups.ElementsAs(ctx, &sgs, false)
		params.SecurityGroups = sgs
	}

	// ----------------------------
	// Call API
	// ----------------------------

	createResp, err := r.client.Highlvlvpc.NewInstance(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Instance Creation Failed", err.Error())
		return
	}

	if createResp == nil {
		resp.Diagnostics.AddError("Invalid API Response", "Nil response from API")
		return
	}

	instanceKrnRaw, ok := (*createResp)["instance_krn"]
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid API Response",
			fmt.Sprintf("Response: %+v", *createResp),
		)
		return
	}

	instanceKrn, ok := instanceKrnRaw.(string)
	if !ok || instanceKrn == "" {
		resp.Diagnostics.AddError(
			"Invalid API Response",
			fmt.Sprintf("Response: %+v", *createResp),
		)
		return
	}


	// -------------------------------------------------
// IMPORTANT: Populate computed fields from CREATE response
// -------------------------------------------------

	if v, ok := (*createResp)["floating_ip_krn"].(string); ok && v != "" {
		plan.FloatingIPKrn = types.StringValue(v)
	} else {
		plan.FloatingIPKrn = types.StringNull()
	}

	if v, ok := (*createResp)["port_krn"].(string); ok && v != "" {
		plan.PortKrn = types.StringValue(v)
	} else {
		plan.PortKrn = types.StringNull()
	}

	if v, ok := (*createResp)["volume_krn"].(string); ok && v != "" {
		plan.VolumeKrn = types.StringValue(v)
	} else {
		plan.VolumeKrn = types.StringNull()
	}

	if v, ok := (*createResp)["floating_ip_address"].(string); ok && v != "" {
		plan.FloatingIPAddress = types.StringValue(v)
	} else {
		plan.FloatingIPAddress = types.StringNull()
	}

	// ----------------------------
	// Wait Until ACTIVE
	// ----------------------------

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var instanceData *krutrim.HighlvlvpcGetInstanceResponse

	for {
		select {
		case <-ctxTimeout.Done():
			resp.Diagnostics.AddError(
				"Timeout",
				"Timed out waiting for instance to become ACTIVE",
			)
			return

		case <-ticker.C:
			rsp, err := r.client.Highlvlvpc.GetInstance(
				ctxTimeout,
				krutrim.HighlvlvpcGetInstanceParams{
					Krn:     instanceKrn,
					XRegion: plan.Region.ValueString(),
				},
			)

			if err != nil || rsp == nil {
				continue
			}

			statusRaw := (*rsp)["status"]
			if status, ok := statusRaw.(string); ok && status == "ACTIVE" {
				if ports, ok := (*rsp)["network_ports"].([]any); ok && len(ports) > 0 {
					instanceData = rsp
					break
				}
			}
		}

		if instanceData != nil {
			break
		}
	}

	if instanceData != nil {

		// KRNs (safe overwrite)
		if plan.FloatingIPKrn.IsNull() {
			if v, ok := (*instanceData)["floating_ip_krn"].(string); ok && v != "" {
				plan.FloatingIPKrn = types.StringValue(v)
			}
		}
	
		if plan.PortKrn.IsNull() {
			if v, ok := (*instanceData)["port_krn"].(string); ok && v != "" {
				plan.PortKrn = types.StringValue(v)
			}
		}
	
		if plan.VolumeKrn.IsNull() {
			if v, ok := (*instanceData)["volume_krn"].(string); ok && v != "" {
				plan.VolumeKrn = types.StringValue(v)
			}
		}
	
		// Metadata
		if v, ok := (*instanceData)["vm_name"].(string); ok && v != "" {
			plan.VMName = types.StringValue(v)
		}
	
		if v, ok := (*instanceData)["status"].(string); ok && v != "" {
			plan.Status = types.StringValue(v)
		}
	
		// Network
		if ports, ok := (*instanceData)["network_ports"].([]any); ok && len(ports) > 0 {
			if port, ok := ports[0].(map[string]any); ok {
	
				if v, ok := port["fixed_ip"].(string); ok && v != "" {
					plan.PrivateIPAddress = types.StringValue(v)
				}
	
				if v, ok := port["floating_ip"].(string); ok && v != "" {
					plan.FloatingIPAddress = types.StringValue(v)
				}
			}
		}
	}
	if plan.VMName.IsNull() {
		plan.VMName = plan.Name
	}

	// ----------------------------
	// Set Terraform State
	// ----------------------------

	plan.ID = types.StringValue(instanceKrn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

/*
=================================================
READ
=================================================
*/

func (r *InstanceResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state InstanceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.Highlvlvpc.GetInstance(
		ctx,
		krutrim.HighlvlvpcGetInstanceParams{
			Krn:     state.ID.ValueString(),
			XRegion: state.Region.ValueString(),
		},
	)

	if err != nil {
		// keep state unless 404 is confirmed
		resp.Diagnostics.AddWarning(
			"Read failed",
			"Could not verify instance existence, keeping state",
		)
		return
	}

	if res == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update computed fields safely


	if v, ok := (*res)["vm_name"].(string); ok {
		state.VMName = types.StringValue(v)
	}
	
	if v, ok := (*res)["status"].(string); ok {
		state.Status = types.StringValue(v)
	}
	
	if ports, ok := (*res)["network_ports"].([]any); ok && len(ports) > 0 {
		if port, ok := ports[0].(map[string]any); ok {
	
			if v, ok := port["fixed_ip"].(string); ok {
				state.PrivateIPAddress = types.StringValue(v)
			}
	
			if v, ok := port["floating_ip"].(string); ok {
				state.FloatingIPAddress = types.StringValue(v)
			}
		}
	}
	if v, ok := (*res)["floating_ip_krn"].(string); ok && v != "" {
		state.FloatingIPKrn = types.StringValue(v)
	}
	
	if v, ok := (*res)["port_krn"].(string); ok && v != "" {
		state.PortKrn = types.StringValue(v)
	}
	
	if v, ok := (*res)["volume_krn"].(string); ok && v != "" {
		state.VolumeKrn = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

/*
=================================================
UPDATE
=================================================
*/

func (r *InstanceResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"Updating an instance is not supported. Terraform will recreate the resource.",
	)
}

/*
=================================================
DELETE
=================================================
*/

func (r *InstanceResource) Delete(
    ctx context.Context,
    req resource.DeleteRequest,
    resp *resource.DeleteResponse,
) {
    var state InstanceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    _, err := r.client.Highlvlvpc.DeleteInstance(
        ctxTimeout,
        state.ID.ValueString(),
        state.DeleteOnTermination.ValueBool(),
    )

    if err != nil {
        resp.Diagnostics.AddWarning(
            "Delete request returned error",
            err.Error(),
        )
    }

    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctxTimeout.Done():
            resp.Diagnostics.AddError(
                "Timeout",
                "Timed out waiting for instance deletion",
            )
            return

        case <-ticker.C:
            _, err := r.client.Highlvlvpc.GetInstance(
                ctxTimeout,
                krutrim.HighlvlvpcGetInstanceParams{
                    Krn:     state.ID.ValueString(),
                    XRegion: state.Region.ValueString(),
                },
            )

            if err != nil {
                resp.State.RemoveResource(ctx)
                return
            }
        }
    }
}