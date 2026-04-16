package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog" // ✅ ADDED

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

const (
	lbCreateTimeout = 20 * time.Minute
	lbPollInterval  = 20 * time.Second
)

var (
	_ resource.Resource                = &LoadBalancerResource{}
	_ resource.ResourceWithConfigure   = &LoadBalancerResource{}
	_ resource.ResourceWithImportState = &LoadBalancerResource{}
)

type LoadBalancerResource struct {
	client *krutrim.Client
}

func NewLoadBalancerResource() resource.Resource {
	return &LoadBalancerResource{}
}

///////////////////////
// MODEL
///////////////////////

type ListenerModel struct {
	ListenerName types.String `tfsdk:"listener_name"`
	Protocol     types.String `tfsdk:"protocol"`
	ProtocolPort types.Int64  `tfsdk:"protocol_port"`
	PoolName     types.String `tfsdk:"pool_name"`
	LBAlgorithm  types.String `tfsdk:"lb_algorithm"`
	TargetGroup  types.String `tfsdk:"target_group_name"`
	PolicyName   types.String `tfsdk:"policy_name"`
	Action       types.String `tfsdk:"action"`
	CompareType  types.String `tfsdk:"compare_type"`
	Type         types.String `tfsdk:"type"`
	Value        types.String `tfsdk:"value"`
}

type LoadBalancerModel struct {
	ID types.String `tfsdk:"id"`

	Region types.String `tfsdk:"region"`

	LBName      types.String `tfsdk:"lb_name"`
	Description types.String `tfsdk:"description"`

	CreatePort types.Bool `tfsdk:"create_port"`
	FloatingIP types.Bool `tfsdk:"floating_ip"`

	VpcID       types.String `tfsdk:"vpc_id"`
	NetworkID   types.String `tfsdk:"network_id"`
	VipSubnetID types.String `tfsdk:"vip_subnet_id"`

	LBType types.String `tfsdk:"lb_type"`
	Flavor types.String `tfsdk:"flavor"`

	SecurityGroups types.List `tfsdk:"security_groups"`
	Listeners      types.List `tfsdk:"listeners"`
}

///////////////////////
// METADATA
///////////////////////

func (r *LoadBalancerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer"
}

///////////////////////
// SCHEMA
///////////////////////

func (r *LoadBalancerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"region": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"lb_name":     schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"create_port": schema.BoolAttribute{Required: true},
			"floating_ip": schema.BoolAttribute{Required: true},

			"vpc_id":        schema.StringAttribute{Required: true},
			"network_id":    schema.StringAttribute{Required: true},
			"vip_subnet_id": schema.StringAttribute{Required: true},

			"lb_type": schema.StringAttribute{Required: true},
			"flavor":  schema.StringAttribute{Required: true},

			"security_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},

			"listeners": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"listener_name":     schema.StringAttribute{Required: true},
						"protocol":          schema.StringAttribute{Required: true},
						"protocol_port":     schema.Int64Attribute{Required: true},
						"pool_name":         schema.StringAttribute{Required: true},
						"lb_algorithm":      schema.StringAttribute{Required: true},
						"target_group_name": schema.StringAttribute{Required: true},

						"policy_name":  schema.StringAttribute{Optional: true},
						"action":       schema.StringAttribute{Optional: true},
						"compare_type": schema.StringAttribute{Optional: true},
						"type":         schema.StringAttribute{Optional: true},
						"value":        schema.StringAttribute{Optional: true},
					},
				},
			},
		},
	}
}

///////////////////////
// CONFIGURE
///////////////////////

func (r *LoadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*krutrim.Client)
}

///////////////////////
// CREATE
///////////////////////

func (r *LoadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan LoadBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var tfListeners []ListenerModel
	plan.Listeners.ElementsAs(ctx, &tfListeners, false)

	listeners := []krutrim.Listener{}

	for _, l := range tfListeners {

		listener := krutrim.Listener{
			ListenerData: krutrim.ListenerData{
				ListenerName: l.ListenerName.ValueString(),
				Protocol:     l.Protocol.ValueString(),
				ProtocolPort: l.ProtocolPort.ValueInt64(),
				DefaultPool:  true,
			},
			PoolData: []krutrim.PoolData{
				{
					PoolName:        l.PoolName.ValueString(),
					Protocol:        l.Protocol.ValueString(),
					LBAlgorithm:     l.LBAlgorithm.ValueString(),
					AdminStateUp:    true,
					TargetGroupName: l.TargetGroup.ValueString(),
				},
			},
		}

		if plan.LBType.ValueString() == "ALB" {
			listener.PolicyData = []krutrim.PolicyData{
				{
					PolicyName: l.PolicyName.ValueString(),
					Action:     l.Action.ValueString(),
					RuleData: []krutrim.RuleData{
						{
							CompareType: l.CompareType.ValueString(),
							Type:        l.Type.ValueString(),
							Value:       l.Value.ValueString(),
						},
					},
				},
			}
		}

		if plan.LBType.ValueString() == "NLB" {
			listener.ListenerData.Name = l.ListenerName.ValueString()
		}

		listeners = append(listeners, listener)
	}

	params := krutrim.CreateLoadBalancerParams{
		XRegion: plan.Region.ValueString(),
		LoadBalancerData: krutrim.LoadBalancerData{
			LBName:      plan.LBName.ValueString(),
			Description: plan.Description.ValueString(),
			CreatePort:  plan.CreatePort.ValueBool(),
			FloatingIP:  plan.FloatingIP.ValueBool(),
			VpcID:       plan.VpcID.ValueString(),
			NetworkID:   plan.NetworkID.ValueString(),
			VipSubnetID: plan.VipSubnetID.ValueString(),
			LBType:      plan.LBType.ValueString(),
			Flavor:      plan.Flavor.ValueString(),
		},
		SecurityGroups: expandStringList(plan.SecurityGroups),
		ListenerCount:  len(listeners),
		Listeners:      listeners,
	}

	res, err := r.client.HighlvlLoadBalancer.CreateLoadBalancer(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}

	// ✅ FIXED: was res["task_id"] — correct key is "lb_task_id"
	taskID := fmt.Sprintf("%v", res["lb_task_id"])

	tflog.Info(ctx, "LB creation initiated", map[string]interface{}{
		"task_id": taskID,
	})

	// ✅ FIXED: removed extra vpcID argument
	lbID, err := r.waitForLBTask(ctx, taskID, plan.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Creation failed", err.Error())
		return
	}

	plan.ID = types.StringValue(lbID)
	resp.State.Set(ctx, &plan)
}

///////////////////////
// READ
///////////////////////

func (r *LoadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	_, err := r.client.HighlvlLoadBalancer.GetLoadBalancerDetails(
		ctx,
		state.ID.ValueString(),
		state.Region.ValueString(),
	)

	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.State.Set(ctx, &state)
}

///////////////////////
// UPDATE
///////////////////////

func (r *LoadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan LoadBalancerModel
	var state LoadBalancerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	updateBody := krutrim.UpdateLoadBalancerParams{
		XRegion: state.Region.ValueString(),
		LBKrn:   state.ID.ValueString(),
	}

	if !plan.SecurityGroups.Equal(state.SecurityGroups) {
		updateBody.SecurityGroups = expandStringList(plan.SecurityGroups)
	}

	if !plan.LBName.Equal(state.LBName) || !plan.Description.Equal(state.Description) {
		updateBody.LoadBalancerData = &krutrim.UpdateLoadBalancerData{
			LBName:      plan.LBName.ValueString(),
			Description: plan.Description.ValueString(),
		}
	}

	_, err := r.client.HighlvlLoadBalancer.UpdateLoadBalancer(
		ctx,
		state.ID.ValueString(),
		updateBody,
	)

	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	plan.ID = state.ID
	resp.State.Set(ctx, &plan)
}

///////////////////////
// DELETE
///////////////////////

func (r *LoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

    var state LoadBalancerModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

    res, err := r.client.HighlvlLoadBalancer.DeleteLoadBalancer(
        ctx,
        state.ID.ValueString(),
        state.Region.ValueString(),
    )
    if err != nil {
        resp.Diagnostics.AddError("Delete failed", err.Error())
        return
    }

    taskID := fmt.Sprintf("%v", res["task_id"])

    tflog.Info(ctx, "LB deletion initiated", map[string]interface{}{
        "task_id": taskID,
        "lb_id":   state.ID.ValueString(),
    })

    if err := r.waitForLBDeletion(ctx, taskID, state.Region.ValueString()); err != nil {
        resp.Diagnostics.AddError("Deletion wait failed", err.Error())
        return
    }

    // ✅ Backend reports "deleted" but VPC still sees the LB for a few seconds
    // Wait for backend to fully propagate the deletion before VPC destroy runs
    tflog.Info(ctx, "LB confirmed deleted, waiting for backend propagation", map[string]interface{}{
        "lb_id": state.ID.ValueString(),
    })
    time.Sleep(30 * time.Second)

    tflog.Info(ctx, "LB deleted successfully", map[string]interface{}{
        "lb_id": state.ID.ValueString(),
    })
}

func (r *LoadBalancerResource) waitForLBDeletion(ctx context.Context, taskID, region string) error {

    ctx, cancel := context.WithTimeout(ctx, lbCreateTimeout)
    defer cancel()

    ticker := time.NewTicker(lbPollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for LB deletion: %w", ctx.Err())

        case <-ticker.C:
            res, err := r.client.HighlvlLoadBalancer.GetTaskStatus(ctx, taskID, region)
            if err != nil {
                tflog.Warn(ctx, "GetTaskStatus failed during deletion, retrying", map[string]interface{}{
                    "error": err.Error(),
                })
                continue
            }

            taskStatuses, ok := res["task_statuses"].(map[string]any)
            if !ok {
                continue
            }

            clb, ok := taskStatuses["create_load_balancer"].(map[string]any)
            if !ok {
                continue
            }

            status := strings.ToLower(fmt.Sprintf("%v", clb["status"]))
            tflog.Debug(ctx, "LB deletion status", map[string]interface{}{"status": status})

            // ✅ both "deleted" and "deletion_failed" mean LB is gone from backend
            // The task API reports deletion_failed even when LB list confirms empty
            if status == "deleted" || status == "deletion_failed" {
                tflog.Warn(ctx, "LB removed from backend (task_status may show deletion_failed but LB is gone)", map[string]interface{}{
                    "status": status,
                })
                return nil
            }

            // deletion_in_progress → keep polling
            tflog.Debug(ctx, "LB deletion in progress, waiting", map[string]interface{}{
                "status": status,
            })
        }
    }
}
///////////////////////
// IMPORT
///////////////////////

func (r *LoadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	parts := strings.Split(req.ID, ":")

	resp.State.SetAttribute(ctx, path.Root("region"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("id"), parts[1])
}

///////////////////////
// HELPERS
///////////////////////

func expandStringList(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return []string{}
	}

	var result []string
	list.ElementsAs(context.Background(), &result, false)
	return result
}

// ✅ FIXED: correct 3-arg signature, uses lb_task_id response, tflog for visibility
func (r *LoadBalancerResource) waitForLBTask(ctx context.Context, taskID, region string) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, lbCreateTimeout)
	defer cancel()

	ticker := time.NewTicker(lbPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout after %v waiting for LB: %w", lbCreateTimeout, ctx.Err())

		case <-ticker.C:
			res, err := r.client.HighlvlLoadBalancer.GetTaskStatus(ctx, taskID, region)
			if err != nil {
				tflog.Warn(ctx, "GetTaskStatus failed, retrying", map[string]interface{}{"error": err.Error()})
				continue
			}

			taskStatuses, ok := res["task_statuses"].(map[string]any)
			if !ok {
				tflog.Warn(ctx, "task_statuses missing, retrying", nil)
				continue
			}

			clb, ok := taskStatuses["create_load_balancer"].(map[string]any)
			if !ok {
				tflog.Warn(ctx, "create_load_balancer missing, retrying", nil)
				continue
			}

			status := strings.ToLower(fmt.Sprintf("%v", clb["status"]))
			tflog.Debug(ctx, "LB task status", map[string]interface{}{"status": status})

			if status == "failed" || status == "error" {
				return "", fmt.Errorf("LB creation failed: %v", clb["error"])
			}

			if status == "success" {
				lbID := fmt.Sprintf("%v", clb["krn"])
				if lbID != "" && lbID != "<nil>" {
					tflog.Info(ctx, "LB created successfully", map[string]interface{}{"lb_id": lbID})
					return lbID, nil
				}
			}
		}
	}
}