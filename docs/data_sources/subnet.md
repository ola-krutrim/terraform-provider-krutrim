---
page_title: "krutrim_subnet Data Source - terraform-provider-krutrim"
subcategory: ""
description: |-
  Retrieves details of an existing Subnet in Krutrim Cloud.
---

# krutrim_subnet (Data Source)

Provides information about an existing subnet inside a VPC.

This data source allows you to fetch subnet details using its ID.

## Example Usage

```hcl
data "krutrim_subnet" "example" {
  id     = "subnet-456"
  vpc_id = "krn:vpc:123"
  region = "In-Bangalore-1"
}
```

## Schema

### Required

- `id` (String) Subnet ID.
- `vpc_id` (String) VPC ID (KRN).
- `region` (String) Region where the subnet exists.

### Read-Only

- `name` (String) Subnet name.
- `description` (String) Subnet description.
- `cidr` (String) CIDR block.
- `gateway_ip` (String) Gateway IP address.
- `ip_version` (String) IP version.
- `ingress` (Boolean) Whether ingress is enabled.
- `egress` (Boolean) Whether egress is enabled.

## Behavior and Usage Notes

- If the subnet does not exist, Terraform will remove it from state.
- Useful for referencing existing infrastructure created outside Terraform.