package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

var (
	_ resource.Resource                = &KKSClusterResource{}
	_ resource.ResourceWithConfigure   = &KKSClusterResource{}
	_ resource.ResourceWithImportState = &KKSClusterResource{}
)

type KKSClusterResource struct {
	client *krutrim.Client
}

type KKSClusterModel struct {
	ID                    types.String `tfsdk:"id"`
	Region                types.String `tfsdk:"region"`
	Name                  types.String `tfsdk:"name"`
	Version               types.String `tfsdk:"version"`
	VpcKRN                types.String `tfsdk:"vpc_krn"`
	SubnetKRNs            types.String `tfsdk:"subnet_krns"`
	EndpointPrivateAccess types.Bool   `tfsdk:"endpoint_private_access"`
	PodIPv4CIDR           types.String `tfsdk:"pod_ipv4_cidr"`
	ServiceIPv4CIDR       types.String `tfsdk:"service_ipv4_cidr"`
	Status                types.String `tfsdk:"status"`
	VersionSupported      types.Bool   `tfsdk:"version_supported"`
	CreatedAt             types.String `tfsdk:"created_at"`
	Kubeconfig            types.String `tfsdk:"kubeconfig"`
}

func NewKKSClusterResource() resource.Resource {
	return &KKSClusterResource{}
}

func (r *KKSClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kks_cluster"
}

func (r *KKSClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Krutrim Kubernetes Service (KKS) cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster KRN.",
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
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Kubernetes version (for example `1.36`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_krn": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_krns": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Subnet KRN string as accepted by the KKS API (not a list).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"endpoint_private_access": schema.BoolAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"pod_ipv4_cidr": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_ipv4_cidr": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"version_supported": schema.BoolAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unix creation timestamp as returned by KKS.",
			},
			"kubeconfig": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Cluster kubeconfig contents. Available after the cluster is PROVISIONED.",
			},
		},
	}
}

func (r *KKSClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KKSClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KKSClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.KKS.V1.Cluster.Create(ctx, krutrim.KKSV1ClusterCreateParams{
		Name:    plan.Name.ValueString(),
		Version: plan.Version.ValueString(),
		XRegion: plan.Region.ValueString(),
		ResourcesVPCConfig: krutrim.KKSResourcesVPCConfig{
			VPCKRN:                plan.VpcKRN.ValueString(),
			SubnetKRNs:            plan.SubnetKRNs.ValueString(),
			EndpointPrivateAccess: plan.EndpointPrivateAccess.ValueBool(),
		},
		KubernetesNetworkConfig: krutrim.KKSKubernetesNetworkConfig{
			PodIPv4CIDR:     plan.PodIPv4CIDR.ValueString(),
			ServiceIPv4CIDR: plan.ServiceIPv4CIDR.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Create KKS Cluster Failed", err.Error())
		return
	}

	clusterKRN := created.Cluster.KRN
	if clusterKRN == "" {
		resp.Diagnostics.AddError("Create KKS Cluster Failed", "API returned an empty cluster KRN")
		return
	}
	plan.ID = types.StringValue(clusterKRN)

	status, err := waitForKKSStatus(ctx, kksCreateTimeout, func(ctx context.Context) (string, error) {
		res, err := r.client.KKS.V1.Cluster.Get(ctx, clusterKRN, krutrim.KKSV1ClusterGetParams{
			XRegion: plan.Region.ValueString(),
		})
		if err != nil {
			return "", err
		}
		return res.Cluster.Status, nil
	}, func(status string) bool {
		return strings.EqualFold(status, "PROVISIONED")
	})
	if err != nil {
		resp.Diagnostics.AddError("KKS Cluster Provisioning Failed", err.Error())
		return
	}

	if err := r.refreshClusterState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Read KKS Cluster Failed", err.Error())
		return
	}
	plan.Status = types.StringValue(status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KKSClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KKSClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refreshClusterState(ctx, &state); err != nil {
		if kksIsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read KKS Cluster Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KKSClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KKSClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Immutable fields use RequiresReplace; keep state aligned with plan.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KKSClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KKSClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.KKS.V1.Cluster.Delete(ctx, state.ID.ValueString(), krutrim.KKSV1ClusterDeleteParams{
		XRegion: state.Region.ValueString(),
	})
	if err != nil && !kksIsNotFound(err) {
		resp.Diagnostics.AddError("Delete KKS Cluster Failed", err.Error())
		return
	}

	if err := waitForKKSGone(ctx, kksDeleteTimeout, func(ctx context.Context) error {
		_, err := r.client.KKS.V1.Cluster.Get(ctx, state.ID.ValueString(), krutrim.KKSV1ClusterGetParams{
			XRegion: state.Region.ValueString(),
		})
		return err
	}); err != nil {
		resp.Diagnostics.AddError("Delete KKS Cluster Failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *KKSClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// region:cluster_krn
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected format: region:cluster_krn")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *KKSClusterResource) refreshClusterState(ctx context.Context, state *KKSClusterModel) error {
	res, err := r.client.KKS.V1.Cluster.Get(ctx, state.ID.ValueString(), krutrim.KKSV1ClusterGetParams{
		XRegion: state.Region.ValueString(),
	})
	if err != nil {
		return err
	}

	c := res.Cluster
	state.Name = types.StringValue(c.Name)
	state.Version = types.StringValue(c.Version)
	state.VpcKRN = types.StringValue(c.ResourcesVPCConfig.VPCKRN)
	state.SubnetKRNs = types.StringValue(c.ResourcesVPCConfig.SubnetKRNs)
	state.EndpointPrivateAccess = types.BoolValue(c.ResourcesVPCConfig.EndpointPrivateAccess)
	state.PodIPv4CIDR = types.StringValue(c.KubernetesNetworkConfig.PodIPv4CIDR)
	state.ServiceIPv4CIDR = types.StringValue(c.KubernetesNetworkConfig.ServiceIPv4CIDR)
	state.Status = types.StringValue(c.Status)
	state.VersionSupported = types.BoolValue(c.VersionSupported)
	state.CreatedAt = types.StringValue(strconv.FormatFloat(c.CreatedAt, 'f', -1, 64))

	if strings.EqualFold(c.Status, "PROVISIONED") {
		kubeconfig, err := r.client.KKS.V1.Cluster.GetKubeconfig(ctx, state.ID.ValueString(), krutrim.KKSV1ClusterGetKubeconfigParams{
			XRegion: state.Region.ValueString(),
		})
		if err != nil {
			if kksIsKubeconfigPending(err) {
				state.Kubeconfig = types.StringNull()
				return nil
			}
			return fmt.Errorf("get kubeconfig: %w", err)
		}
		state.Kubeconfig = types.StringValue(string(kubeconfig))
	} else {
		state.Kubeconfig = types.StringNull()
	}

	return nil
}

// kksIsKubeconfigPending reports errors that can occur while a newly
// provisioned cluster's kubeconfig is still being replicated. These are
// transient, so callers can retain the cluster and let a later refresh retry.
func kksIsKubeconfigPending(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *krutrim.Error
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusServiceUnavailable {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "still being provisioned") ||
		strings.Contains(message, "unable to download kubeconfig")
}
