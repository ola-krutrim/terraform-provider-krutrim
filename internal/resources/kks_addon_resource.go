package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var (
	_ resource.Resource                = &KKSAddonResource{}
	_ resource.ResourceWithConfigure   = &KKSAddonResource{}
	_ resource.ResourceWithImportState = &KKSAddonResource{}
)

type KKSAddonResource struct {
	client *krutrim.Client
}

type KKSAddonModel struct {
	ID           types.String `tfsdk:"id"`
	Region       types.String `tfsdk:"region"`
	ClusterKRN   types.String `tfsdk:"cluster_krn"`
	AddonName    types.String `tfsdk:"addon_name"`
	AddonVersion types.String `tfsdk:"addon_version"`
	Status       types.String `tfsdk:"status"`
	ClusterName  types.String `tfsdk:"cluster_name"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func NewKKSAddonResource() resource.Resource {
	return &KKSAddonResource{}
}

func (r *KKSAddonResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kks_addon"
}

func (r *KKSAddonResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a KKS add-on installed on a cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Installed add-on KRN.",
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
			"addon_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"addon_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Omit to use the server default version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"cluster_name": schema.StringAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *KKSAddonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KKSAddonResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KKSAddonModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := krutrim.KKSV1ClusterAddonCreateParams{
		AddonName: plan.AddonName.ValueString(),
		XRegion:   plan.Region.ValueString(),
	}
	if !plan.AddonVersion.IsNull() && !plan.AddonVersion.IsUnknown() && plan.AddonVersion.ValueString() != "" {
		params.AddonVersion = krutrim.String(plan.AddonVersion.ValueString())
	}

	created, err := r.client.KKS.V1.Cluster.Addon.Create(ctx, plan.ClusterKRN.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Create KKS Addon Failed", err.Error())
		return
	}

	addonKRN := created.Addon.AddonKRN
	if addonKRN == "" {
		resp.Diagnostics.AddError("Create KKS Addon Failed", "API returned an empty addon KRN")
		return
	}
	plan.ID = types.StringValue(addonKRN)
	plan.AddonName = types.StringValue(created.Addon.AddonName)
	plan.AddonVersion = types.StringValue(created.Addon.AddonVersion)
	plan.Status = types.StringValue(created.Addon.Status)
	plan.ClusterName = types.StringValue(created.Addon.ClusterName)
	plan.CreatedAt = types.StringValue(created.Addon.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))

	status, err := waitForKKSStatus(ctx, kksCreateTimeout, func(ctx context.Context) (string, error) {
		summary, err := r.findInstalledAddon(ctx, plan.ClusterKRN.ValueString(), plan.Region.ValueString(), addonKRN)
		if err != nil {
			return "", err
		}
		return summary.Status, nil
	}, func(status string) bool {
		return !kksIsPending(status) && !kksIsTerminalFailure(status)
	})
	if err != nil {
		resp.Diagnostics.AddError("KKS Addon Provisioning Failed", err.Error())
		return
	}
	plan.Status = types.StringValue(status)

	if err := r.refreshAddonState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Read KKS Addon Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KKSAddonResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KKSAddonModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refreshAddonState(ctx, &state); err != nil {
		if kksIsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read KKS Addon Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KKSAddonResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KKSAddonModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KKSAddonResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KKSAddonModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.KKS.V1.Cluster.Addon.Delete(ctx, state.ClusterKRN.ValueString(), state.ID.ValueString(), krutrim.KKSV1ClusterAddonDeleteParams{
		XRegion: state.Region.ValueString(),
	})
	if err != nil && !kksIsNotFound(err) {
		resp.Diagnostics.AddError("Delete KKS Addon Failed", err.Error())
		return
	}

	if err := waitForKKSGone(ctx, kksDeleteTimeout, func(ctx context.Context) error {
		_, err := r.findInstalledAddon(ctx, state.ClusterKRN.ValueString(), state.Region.ValueString(), state.ID.ValueString())
		return err
	}); err != nil {
		resp.Diagnostics.AddError("Delete KKS Addon Failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *KKSAddonResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// region:cluster_krn:addon_krn
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected format: region:cluster_krn:addon_krn")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_krn"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (r *KKSAddonResource) findInstalledAddon(ctx context.Context, clusterKRN, region, addonKRN string) (*krutrim.KKSAddonSummary, error) {
	listed, err := r.client.KKS.V1.Cluster.Addon.List(ctx, clusterKRN, krutrim.KKSV1ClusterAddonListParams{
		XRegion: region,
	})
	if err != nil {
		return nil, err
	}
	for i := range listed.Addons {
		if listed.Addons[i].AddonKRN == addonKRN {
			return &listed.Addons[i], nil
		}
	}
	return nil, kksNotFoundError{resource: "addon"}
}

func (r *KKSAddonResource) refreshAddonState(ctx context.Context, state *KKSAddonModel) error {
	summary, err := r.findInstalledAddon(ctx, state.ClusterKRN.ValueString(), state.Region.ValueString(), state.ID.ValueString())
	if err != nil {
		return err
	}

	state.AddonName = types.StringValue(summary.AddonName)
	state.AddonVersion = types.StringValue(summary.AddonVersion)
	state.Status = types.StringValue(summary.Status)
	if state.ClusterName.IsNull() || state.ClusterName.IsUnknown() {
		state.ClusterName = types.StringNull()
	}
	state.CreatedAt = types.StringValue(fmt.Sprintf("%v", summary.CreatedAt))
	return nil
}
