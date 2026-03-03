---
page_title: "krutrim_security_group_rule Resource - terraform-provider-krutrim"
subcategory: ""
description: |-
  Manages a Krutrim Security Group Rule.
  This resource allows you to create and attach ingress or egress rules
  to an existing security group inside a VPC.
---

# krutrim_security_group_rule (Resource)

Manages a Krutrim Security Group Rule.

Security group rules define inbound (ingress) or outbound (egress) traffic policies
for a security group.

When applied:
1. A rule is created.
2. The rule is attached to the specified security group.

When destroyed:
- The rule is detached from the security group.

## Example Usage

```hcl
resource "krutrim_security_group_rule" "allow_http" {
  security_group_id = "krn:sg:123"
  vpc_id            = "krn:vpc:123"
  region            = "In-Bangalore-1"

  description       = "Allow HTTP traffic"
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_min_range    = 80
  port_max_range    = 80
  remote_ip_prefix  = "0.0.0.0/0"
}
```

## Schema

### Required

- `security_group_id` (String) Security Group ID (KRN) to attach this rule to.
- `vpc_id` (String) VPC ID (KRN).
- `direction` (String) Direction of the rule: `"ingress"` or `"egress"`.
- `ethertype` (String) Ethertype: `"IPv4"` or `"IPv6"`.
- `protocol` (String) Protocol (`"tcp"`, `"udp"`, `"icmp"`, etc.).
- `region` (String) Region (e.g., `"In-Bangalore-1"`).

### Optional

- `description` (String) Rule description.
- `port_min_range` (Number) Minimum port number.
- `port_max_range` (Number) Maximum port number.
- `remote_ip_prefix` (String) Remote IP prefix in CIDR notation (e.g., `"0.0.0.0/0"`).

### Computed

- `id` (String) Rule ID.

## Behavior and Usage Notes

- Rules are created first and then attached to the specified security group.
- Rules cannot be updated; changing attributes requires recreation.
- If the rule is detached or deleted outside Terraform, it will be removed from state during refresh.
- Deleting the resource detaches the rule from the security group.