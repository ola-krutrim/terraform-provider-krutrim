package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var (
	_ resource.Resource                = &KKSNodeGroupResource{}
	_ resource.ResourceWithConfigure   = &KKSNodeGroupResource{}
	_ resource.ResourceWithImportState = &KKSNodeGroupResource{}
)

type KKSNodeGroupResource struct {
	client *krutrim.Client
}

type KKSScalingConfigModel struct {
	MinSize     types.Int64 `tfsdk:"min_size"`
	MaxSize     types.Int64 `tfsdk:"max_size"`
	DesiredSize types.Int64 `tfsdk:"desired_size"`
}

type KKSTaintModel struct {
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
	Effect types.String `tfsdk:"effect"`
}

type KKSRemoteAccessModel struct {
	SSHKeyKRN                types.String `tfsdk:"ssh_key_krn"`
	SourceSecurityGroupsKRNs types.List   `tfsdk:"source_security_groups_krns"`
}

type KKSNodeGroupModel struct {
	ID                types.String           `tfsdk:"id"`
	Region            types.String           `tfsdk:"region"`
	ClusterKRN        types.String           `tfsdk:"cluster_krn"`
	Name              types.String           `tfsdk:"name"`
	InstanceTypes     types.String           `tfsdk:"instance_types"`
	DiskSize          types.Int64            `tfsdk:"disk_size"`
	SubnetsKRN        types.String           `tfsdk:"subnets_krn"`
	ScalingConfig     *KKSScalingConfigModel `tfsdk:"scaling_config"`
	Labels            types.Map              `tfsdk:"labels"`
	Taints            []KKSTaintModel        `tfsdk:"taints"`
	RemoteAccess      *KKSRemoteAccessModel  `tfsdk:"remote_access"`
	NodeRepairEnabled types.Bool             `tfsdk:"node_repair_enabled"`
	Status            types.String           `tfsdk:"status"`
	Version           types.String           `tfsdk:"version"`
	CreatedAt         types.String           `tfsdk:"created_at"`
}

func NewKKSNodeGroupResource() resource.Resource {
	return &KKSNodeGroupResource{}
}

func (r *KKSNodeGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kks_node_group"
}

func (r *KKSNodeGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a KKS node group within a cluster. Scaling can be updated in place; other attributes force replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Node group KRN.",
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
			"cluster_krn": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_types": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Instance flavor name string (for example `CPU-4x-16GB`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disk_size": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"subnets_krn": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scaling_config": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"min_size":     schema.Int64Attribute{Required: true},
					"max_size":     schema.Int64Attribute{Required: true},
					"desired_size": schema.Int64Attribute{Required: true},
				},
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"taints": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":    schema.StringAttribute{Required: true},
						"value":  schema.StringAttribute{Required: true},
						"effect": schema.StringAttribute{Required: true},
					},
				},
			},
			"remote_access": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"ssh_key_krn": schema.StringAttribute{Optional: true},
					"source_security_groups_krns": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			"node_repair_enabled": schema.BoolAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"version": schema.StringAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *KKSNodeGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *krutrim.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *KKSNodeGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KKSNodeGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.ScalingConfig == nil {
		resp.Diagnostics.AddError("Invalid Configuration", "scaling_config is required")
		return
	}

	params, diags := buildNodeGroupCreateParams(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.KKS.V1.Cluster.NodeGroup.Create(ctx, plan.ClusterKRN.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Create KKS Node Group Failed", err.Error())
		return
	}

	nodeGroupKRN := created.Nodegroup.KRN
	if nodeGroupKRN == "" {
		resp.Diagnostics.AddError("Create KKS Node Group Failed", "API returned an empty node group KRN")
		return
	}
	plan.ID = types.StringValue(nodeGroupKRN)

	status, err := waitForKKSStatus(ctx, kksCreateTimeout, func(ctx context.Context) (string, error) {
		res, err := r.client.KKS.V1.Cluster.NodeGroup.Get(ctx, plan.ClusterKRN.ValueString(), nodeGroupKRN, krutrim.KKSV1ClusterNodeGroupGetParams{
			XRegion: plan.Region.ValueString(),
		})
		if err != nil {
			return "", err
		}
		return res.Nodegroup.Status, nil
	}, func(status string) bool {
		return !kksIsPending(status) && !kksIsTerminalFailure(status)
	})
	if err != nil {
		resp.Diagnostics.AddError("KKS Node Group Provisioning Failed", err.Error())
		return
	}

	if err := r.refreshNodeGroupState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Read KKS Node Group Failed", err.Error())
		return
	}
	plan.Status = types.StringValue(status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KKSNodeGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KKSNodeGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refreshNodeGroupState(ctx, &state); err != nil {
		if kksIsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read KKS Node Group Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KKSNodeGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KKSNodeGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.ScalingConfig == nil {
		resp.Diagnostics.AddError("Invalid Configuration", "scaling_config is required")
		return
	}

	_, err := r.client.KKS.V1.Cluster.NodeGroup.UpdateConfig(ctx, plan.ClusterKRN.ValueString(), plan.ID.ValueString(), krutrim.KKSV1ClusterNodeGroupUpdateConfigParams{
		XRegion: plan.Region.ValueString(),
		ScalingConfig: krutrim.KKSScalingConfig{
			MinSize:     plan.ScalingConfig.MinSize.ValueInt64(),
			MaxSize:     plan.ScalingConfig.MaxSize.ValueInt64(),
			DesiredSize: plan.ScalingConfig.DesiredSize.ValueInt64(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Update KKS Node Group Failed", err.Error())
		return
	}

	_, err = waitForKKSStatus(ctx, kksCreateTimeout, func(ctx context.Context) (string, error) {
		res, err := r.client.KKS.V1.Cluster.NodeGroup.Get(ctx, plan.ClusterKRN.ValueString(), plan.ID.ValueString(), krutrim.KKSV1ClusterNodeGroupGetParams{
			XRegion: plan.Region.ValueString(),
		})
		if err != nil {
			return "", err
		}
		return res.Nodegroup.Status, nil
	}, func(status string) bool {
		return !kksIsPending(status) && !kksIsTerminalFailure(status)
	})
	if err != nil {
		resp.Diagnostics.AddError("KKS Node Group Update Failed", err.Error())
		return
	}

	if err := r.refreshNodeGroupState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Read KKS Node Group Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KKSNodeGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KKSNodeGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.KKS.V1.Cluster.NodeGroup.Delete(ctx, state.ClusterKRN.ValueString(), state.ID.ValueString(), krutrim.KKSV1ClusterNodeGroupDeleteParams{
		XRegion: state.Region.ValueString(),
	})
	if err != nil && !kksIsNotFound(err) {
		resp.Diagnostics.AddError("Delete KKS Node Group Failed", err.Error())
		return
	}

	if err := waitForKKSGone(ctx, kksDeleteTimeout, func(ctx context.Context) error {
		_, err := r.client.KKS.V1.Cluster.NodeGroup.Get(ctx, state.ClusterKRN.ValueString(), state.ID.ValueString(), krutrim.KKSV1ClusterNodeGroupGetParams{
			XRegion: state.Region.ValueString(),
		})
		return err
	}); err != nil {
		resp.Diagnostics.AddError("Delete KKS Node Group Failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *KKSNodeGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// region:cluster_krn:node_group_krn
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected format: region:cluster_krn:node_group_krn")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_krn"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func buildNodeGroupCreateParams(ctx context.Context, plan KKSNodeGroupModel) (krutrim.KKSV1ClusterNodeGroupCreateParams, diag.Diagnostics) {
	var diags diag.Diagnostics

	params := krutrim.KKSV1ClusterNodeGroupCreateParams{
		Name:          plan.Name.ValueString(),
		InstanceTypes: plan.InstanceTypes.ValueString(),
		DiskSize:      plan.DiskSize.ValueInt64(),
		SubnetsKRN:    plan.SubnetsKRN.ValueString(),
		XRegion:       plan.Region.ValueString(),
		ScalingConfig: krutrim.KKSScalingConfig{
			MinSize:     plan.ScalingConfig.MinSize.ValueInt64(),
			MaxSize:     plan.ScalingConfig.MaxSize.ValueInt64(),
			DesiredSize: plan.ScalingConfig.DesiredSize.ValueInt64(),
		},
	}

	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		labels := map[string]string{}
		diags.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		params.Labels = labels
	}

	if plan.Taints != nil {
		taints := make([]krutrim.KKSTaint, 0, len(plan.Taints))
		for _, t := range plan.Taints {
			taints = append(taints, krutrim.KKSTaint{
				Key:    t.Key.ValueString(),
				Value:  t.Value.ValueString(),
				Effect: t.Effect.ValueString(),
			})
		}
		params.Taints = taints
	}

	if plan.RemoteAccess != nil {
		ra := &krutrim.KKSRemoteAccess{
			SSHKeyKRN: plan.RemoteAccess.SSHKeyKRN.ValueString(),
		}
		if !plan.RemoteAccess.SourceSecurityGroupsKRNs.IsNull() && !plan.RemoteAccess.SourceSecurityGroupsKRNs.IsUnknown() {
			var groups []string
			diags.Append(plan.RemoteAccess.SourceSecurityGroupsKRNs.ElementsAs(ctx, &groups, false)...)
			ra.SourceSecurityGroupsKRNs = groups
		}
		params.RemoteAccess = ra
	}

	if !plan.NodeRepairEnabled.IsNull() && !plan.NodeRepairEnabled.IsUnknown() {
		params.NodeRepairConfig = &krutrim.KKSNodeRepairConfig{
			Enabled: plan.NodeRepairEnabled.ValueBool(),
		}
	}

	return params, diags
}

func (r *KKSNodeGroupResource) refreshNodeGroupState(ctx context.Context, state *KKSNodeGroupModel) error {
	res, err := r.client.KKS.V1.Cluster.NodeGroup.Get(ctx, state.ClusterKRN.ValueString(), state.ID.ValueString(), krutrim.KKSV1ClusterNodeGroupGetParams{
		XRegion: state.Region.ValueString(),
	})
	if err != nil {
		return err
	}

	ng := res.Nodegroup
	state.Name = types.StringValue(ng.Name)
	state.InstanceTypes = types.StringValue(ng.InstanceTypes)
	state.DiskSize = types.Int64Value(ng.DiskSize)
	state.SubnetsKRN = types.StringValue(ng.SubnetsKRN)
	state.Status = types.StringValue(ng.Status)
	state.Version = types.StringValue(ng.Version)
	state.CreatedAt = types.StringValue(strconv.FormatFloat(ng.CreatedAt, 'f', -1, 64))
	state.ScalingConfig = &KKSScalingConfigModel{
		MinSize:     types.Int64Value(ng.ScalingConfig.MinSize),
		MaxSize:     types.Int64Value(ng.ScalingConfig.MaxSize),
		DesiredSize: types.Int64Value(ng.ScalingConfig.DesiredSize),
	}

	if ng.Labels != nil {
		elems := make(map[string]attr.Value, len(ng.Labels))
		for k, v := range ng.Labels {
			elems[k] = types.StringValue(v)
		}
		state.Labels = types.MapValueMust(types.StringType, elems)
	} else if state.Labels.IsNull() {
		state.Labels = types.MapNull(types.StringType)
	}

	if ng.Taints != nil {
		taints := make([]KKSTaintModel, 0, len(ng.Taints))
		for _, t := range ng.Taints {
			taints = append(taints, KKSTaintModel{
				Key:    types.StringValue(t.Key),
				Value:  types.StringValue(t.Value),
				Effect: types.StringValue(t.Effect),
			})
		}
		state.Taints = taints
	}

	state.NodeRepairEnabled = types.BoolValue(ng.NodeRepairConfig.Enabled)

	if ng.RemoteAccess.SSHKeyKRN != "" || len(ng.RemoteAccess.SourceSecurityGroupsKRNs) > 0 {
		sgValues := make([]attr.Value, 0, len(ng.RemoteAccess.SourceSecurityGroupsKRNs))
		for _, sg := range ng.RemoteAccess.SourceSecurityGroupsKRNs {
			sgValues = append(sgValues, types.StringValue(sg))
		}
		state.RemoteAccess = &KKSRemoteAccessModel{
			SSHKeyKRN:                types.StringValue(ng.RemoteAccess.SSHKeyKRN),
			SourceSecurityGroupsKRNs: types.ListValueMust(types.StringType, sgValues),
		}
	}

	return nil
}
