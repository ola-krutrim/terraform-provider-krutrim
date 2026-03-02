package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
	"github.com/ola-krutrim/krutrim-go-sdk/packages/param" // Add this import
)

type SecurityGroupRuleModel struct {
	ID              types.String `tfsdk:"id"`
	SecurityGroupID types.String `tfsdk:"security_group_id"`
	VpcID           types.String `tfsdk:"vpc_id"`
	Description     types.String `tfsdk:"description"`
	Direction       types.String `tfsdk:"direction"`
	Ethertype       types.String `tfsdk:"ethertype"`
	Protocol        types.String `tfsdk:"protocol"`
	PortMinRange    types.Int64  `tfsdk:"port_min_range"`
	PortMaxRange    types.Int64  `tfsdk:"port_max_range"`
	RemoteIPPrefix  types.String `tfsdk:"remote_ip_prefix"`
	Region          types.String `tfsdk:"region"`
}

type SecurityGroupRuleResource struct {
	client *krutrim.Client
}

func NewSecurityGroupRuleResource() resource.Resource {
	return &SecurityGroupRuleResource{}
}

func (r *SecurityGroupRuleResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_security_group_rule"
}

func (r *SecurityGroupRuleResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Manages a Krutrim Security Group Rule",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Rule ID",
			},
			"security_group_id": schema.StringAttribute{
				Required:    true,
				Description: "Security Group ID (KRN) to attach this rule to",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "VPC ID (KRN)",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Rule description",
			},
			"direction": schema.StringAttribute{
				Required:    true,
				Description: "Direction: 'ingress' or 'egress'",
			},
			"ethertype": schema.StringAttribute{
				Required:    true,
				Description: "Ethertype: 'IPv4' or 'IPv6'",
			},
			"protocol": schema.StringAttribute{
				Required:    true,
				Description: "Protocol: 'tcp', 'udp', 'icmp', etc.",
			},
			"port_min_range": schema.Int64Attribute{
				Optional:    true,
				Description: "Minimum port number",
			},
			"port_max_range": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum port number",
			},
			"remote_ip_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Remote IP prefix (CIDR notation)",
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "Region (e.g., In-Bangalore-1)",
			},
		},
	}
}

func (r *SecurityGroupRuleResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	_ *resource.ConfigureResponse,
) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*krutrim.Client)
	}
}

func (r *SecurityGroupRuleResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SecurityGroupRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Create the rule
	createParams := krutrim.SecurityGroupV1NewRuleParams{
		Direction:  krutrim.SecurityGroupV1NewRuleParamsDirection(plan.Direction.ValueString()),
		Ethertypes: krutrim.SecurityGroupV1NewRuleParamsEthertypes(plan.Ethertype.ValueString()),
		Protocol:   plan.Protocol.ValueString(),
		Vpcid:      plan.VpcID.ValueString(),
		XRegion:    plan.Region.ValueString(),
	}

	// Set optional description
	if !plan.Description.IsNull() && plan.Description.ValueString() != "" {
		createParams.Description = param.NewOpt(plan.Description.ValueString())
	}
	if !plan.PortMinRange.IsNull() {
		createParams.PortMinRange = plan.PortMinRange.ValueInt64()
	}
	if !plan.PortMaxRange.IsNull() {
		createParams.PortMaxRange = plan.PortMaxRange.ValueInt64()
	}
	if !plan.RemoteIPPrefix.IsNull() && plan.RemoteIPPrefix.ValueString() != "" {
		createParams.RemoteIPPrefix = plan.RemoteIPPrefix.ValueString()
	}

	rule, err := r.client.SecurityGroup.V1.NewRule(ctx, createParams) // Changed to .V1.NewRule
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating security group rule",
			"Could not create rule: "+err.Error(),
		)
		return
	}

	if rule.ID == "" {
		raw := rule.RawJSON()

		var env map[string]any
		_ = json.Unmarshal([]byte(raw), &env)

		if res, ok := env["result"].(map[string]any); ok {
			if id, ok := res["id"].(string); ok {
				plan.ID = types.StringValue(id)
			}
		}
	} else {
		plan.ID = types.StringValue(rule.ID)
	}



	if rule.Description != "" {
		plan.Description = types.StringValue(rule.Description)
	}

	// Step 2: Attach the rule to the security group
	attachParams := krutrim.SecurityGroupV1AttachRuleParams{
		Ruleid:     plan.ID.ValueString(),
		Securityid: plan.SecurityGroupID.ValueString(),
		Vpcid:      plan.VpcID.ValueString(),
		XRegion:    plan.Region.ValueString(),
	}

	_, err = r.client.SecurityGroup.V1.AttachRule(ctx, attachParams) // Changed to .V1.AttachRule
	if err != nil {
		resp.Diagnostics.AddError(
			"Error attaching rule to security group",
			"Could not attach rule: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupRuleResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SecurityGroupRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := krutrim.SecurityGroupV1ListParams{
		XRegion: state.Region.ValueString(),
	}

	sgList, err := r.client.SecurityGroup.V1.List(
		ctx,
		state.VpcID.ValueString(),
		params,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading security groups",
			err.Error(),
		)
		return
	}

	// 🔑 Parse real API response (result[])
	raw := sgList.RawJSON()

	var decoded struct {
		Result []struct {
			ID    string   `json:"id"`
			Rules []string `json:"rules"`
		} `json:"result"`
	}

	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		resp.Diagnostics.AddError(
			"Error parsing security group list response",
			err.Error(),
		)
		return
	}

	// 🔍 Find SG and check rule attachment
	for _, sg := range decoded.Result {
		if sg.ID == state.SecurityGroupID.ValueString() {
			for _, ruleID := range sg.Rules {
				if ruleID == state.ID.ValueString() {
					// ✅ Rule still attached → keep state
					resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
					return
				}
			}

			// SG exists but rule missing
			resp.Diagnostics.AddWarning(
				"Security Group Rule Not Found",
				fmt.Sprintf(
					"Rule %s is no longer attached to security group %s. Removing from state.",
					state.ID.ValueString(),
					state.SecurityGroupID.ValueString(),
				),
			)
			resp.State.RemoveResource(ctx)
			return
		}
	}

	// SG itself missing
	resp.Diagnostics.AddWarning(
		"Security Group Not Found",
		fmt.Sprintf(
			"Security group %s was deleted. Removing rule %s from state.",
			state.SecurityGroupID.ValueString(),
			state.ID.ValueString(),
		),
	)
	resp.State.RemoveResource(ctx)
}


func (r *SecurityGroupRuleResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	// Rules cannot be updated - require recreation
	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"Security group rules cannot be updated. Use terraform apply with -replace flag.",
	)
}

func (r *SecurityGroupRuleResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SecurityGroupRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Detach the rule from the security group
	params := krutrim.SecurityGroupV1DetachRuleParams{
		Ruleid:     state.ID.ValueString(),
		Securityid: state.SecurityGroupID.ValueString(),
		Vpcid:      state.VpcID.ValueString(),
		XRegion:    state.Region.ValueString(),
	}

	_, err := r.client.SecurityGroup.V1.DetachRule(ctx, params) // Changed to .V1.DetachRule
	if err != nil {
		errString := err.Error()
		if strings.Contains(errString, "404") || strings.Contains(errString, "not found") {
			// Already detached or deleted - just remove from state
			resp.Diagnostics.AddWarning(
				"Rule Already Detached",
				"Rule was already detached or deleted outside Terraform.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error detaching rule from security group",
			"Could not detach rule: "+err.Error(),
		)
		return
	}
}
