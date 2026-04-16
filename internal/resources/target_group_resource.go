package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var (
	_ resource.Resource              = &TargetGroupResource{}
	_ resource.ResourceWithConfigure = &TargetGroupResource{}
)

type TargetGroupResource struct {
	client *krutrim.Client
}

func NewTargetGroupResource() resource.Resource {
	return &TargetGroupResource{}
}

///////////////////////
// MODEL
///////////////////////

type MemberModel struct {
	Name         types.String `tfsdk:"name"`
	Address      types.String `tfsdk:"address"`
	ProtocolPort types.Int64  `tfsdk:"protocol_port"`
	Weight       types.Int64  `tfsdk:"weight"`
}

type HealthMonitorModel struct {
	Type       types.String `tfsdk:"type"`
	Delay      types.Int64  `tfsdk:"delay"`
	Timeout    types.Int64  `tfsdk:"timeout"`
	MaxRetries types.Int64  `tfsdk:"max_retries"`
	URLPath    types.String `tfsdk:"url_path"`
}

type TargetGroupModel struct {
	ID types.String `tfsdk:"id"`

	Region types.String `tfsdk:"region"`
	VpcID  types.String `tfsdk:"vpc_id"`

	Name types.String `tfsdk:"name"`

	Members types.List   `tfsdk:"members"`
	Health  types.Object `tfsdk:"health_monitor"`
}

///////////////////////
// METADATA
///////////////////////

func (r *TargetGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target_group"
}

///////////////////////
// SCHEMA
///////////////////////

func (r *TargetGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"region": schema.StringAttribute{Required: true},
			"vpc_id": schema.StringAttribute{Required: true},
			"name":   schema.StringAttribute{Required: true},

			"members": schema.ListNestedAttribute{
				Required: true, // ✅ enforce at least one member
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Required: true},
						"address": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},	
						"protocol_port": schema.Int64Attribute{
							Required: true,
						},
						"weight": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},

			"health_monitor": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type":        schema.StringAttribute{Required: true},
					"delay":       schema.Int64Attribute{Required: true},
					"timeout":     schema.Int64Attribute{Required: true},
					"max_retries": schema.Int64Attribute{Required: true},
					"url_path":    schema.StringAttribute{Optional: true},
				},
			},
		},
	}
}

///////////////////////
// CONFIGURE
///////////////////////

func (r *TargetGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*krutrim.Client)
}

///////////////////////
// CREATE
///////////////////////

func (r *TargetGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan TargetGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	// ✅ Validate members
	if plan.Members.IsNull() || plan.Members.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing members",
			"At least one member must be provided",
		)
		return
	}

	members := expandMembers(ctx, plan.Members)

	// Expand health monitor
	var hm HealthMonitorModel
	plan.Health.As(ctx, &hm, basetypes.ObjectAsOptions{})

	params := krutrim.CreateTargetGroupParams{
		XRegion:         plan.Region.ValueString(),
		VpcID:           plan.VpcID.ValueString(),
		TargetGroupName: plan.Name.ValueString(),
		Members:         members,
		HealthMonitor: krutrim.HealthMonitor{
			Name:       "hm-" + plan.Name.ValueString(),
			HType:      hm.Type.ValueString(),
			Delay:      hm.Delay.ValueInt64(),
			Timeout:    hm.Timeout.ValueInt64(),
			MaxRetries: hm.MaxRetries.ValueInt64(),
			URLPath:    hm.URLPath.ValueString(),
		},
	}

	_, err := r.client.HighlvlLoadBalancer.CreateTargetGroup(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Create Target Group failed", err.Error())
		return
	}

	// ✅ Strong ID
	plan.ID = types.StringValue(
		fmt.Sprintf("%s/%s", plan.VpcID.ValueString(), plan.Name.ValueString()),
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

///////////////////////
// READ
///////////////////////

func (r *TargetGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state TargetGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	_, err := r.client.HighlvlLoadBalancer.GetTargetGroup(ctx, krutrim.GetTargetGroupParams{
		XRegion:         state.Region.ValueString(),
		VpcID:           state.VpcID.ValueString(),
		TargetGroupName: state.Name.ValueString(),
	})

	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// ✅ keep state (can enhance later with full mapping)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

///////////////////////
// UPDATE
///////////////////////

func (r *TargetGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan TargetGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	members := expandMembers(ctx, plan.Members)

	var hm HealthMonitorModel
	plan.Health.As(ctx, &hm, basetypes.ObjectAsOptions{})

	params := krutrim.UpdateTargetGroupParams{
		XRegion:         plan.Region.ValueString(),
		VpcID:           plan.VpcID.ValueString(),
		TargetGroupName: plan.Name.ValueString(),
		Members:         members,
		HealthMonitor: &krutrim.HealthMonitor{
			Name:       "hm-" + plan.Name.ValueString(),
			HType:      hm.Type.ValueString(),
			Delay:      hm.Delay.ValueInt64(),
			Timeout:    hm.Timeout.ValueInt64(),
			MaxRetries: hm.MaxRetries.ValueInt64(),
			URLPath:    hm.URLPath.ValueString(),
		},
	}

	_, err := r.client.HighlvlLoadBalancer.UpdateTargetGroup(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	plan.ID = types.StringValue(
		fmt.Sprintf("%s/%s", plan.VpcID.ValueString(), plan.Name.ValueString()),
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

///////////////////////
// DELETE
///////////////////////

func (r *TargetGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state TargetGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	_, err := r.client.HighlvlLoadBalancer.DeleteTargetGroup(
		ctx,
		state.VpcID.ValueString(),
		state.Name.ValueString(),
		state.Region.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}

///////////////////////
// HELPERS
///////////////////////

func expandMembers(ctx context.Context, list types.List) []krutrim.TargetGroupMember {

	var membersModel []MemberModel
	list.ElementsAs(ctx, &membersModel, false)

	var members []krutrim.TargetGroupMember

	for _, m := range membersModel {
		members = append(members, krutrim.TargetGroupMember{
			Name:         m.Name.ValueString(),
			Address:      m.Address.ValueString(),
			ProtocolPort: m.ProtocolPort.ValueInt64(),
			Weight:       m.Weight.ValueInt64(),
		})
	}

	return members
}