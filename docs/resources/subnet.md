---
page_title: "krutrim_subnet Resource - terraform-provider-krutrim"
subcategory: ""
description: |-
  Manages a Subnet resource in Krutrim Cloud.
  This resource allows you to create and manage subnets inside a VPC network.
---

# krutrim_subnet (Resource)

Manages a Subnet resource.

Subnets define IP address ranges inside a VPC and are used to deploy instances.

When applied:
- A new subnet is created inside the specified VPC.

When destroyed:
- The subnet is deleted (unless it is a primary subnet).

## Example Usage

```hcl
resource "krutrim_subnet" "example" {
  region     = "In-Bangalore-1"
  vpc_id     = "krn:vpc:123"
  network_id = "krn:network:123"

  name        = "app-subnet"
  description = "Subnet for application servers"

  cidr       = "10.0.1.0/24"
  gateway_ip = "10.0.1.1"

  ip_version = "4"
  ingress    = true
  egress     = true
}
```

## Schema

### Required

- `region` (String) Region where the subnet will be created.
- `vpc_id` (String) VPC ID (KRN).
- `network_id` (String) Network ID (KRN).
- `name` (String) Subnet name.
- `cidr` (String) CIDR block for the subnet (e.g., `"10.0.1.0/24"`).
- `gateway_ip` (String) Gateway IP address inside the subnet.

### Optional

- `description` (String, default: `""`) Subnet description.
- `ip_version` (String, default: `"4"`) IP version (`"4"` or `"6"`).
- `ingress` (Boolean, default: `true`) Enable inbound traffic.
- `egress` (Boolean, default: `true`) Enable outbound traffic.

### Computed

- `id` (String) Subnet ID.

## Import

Subnet can be imported using:

```
terraform import krutrim_subnet.example region:vpc_id:subnet_id
```

Example:

```
terraform import krutrim_subnet.example In-Bangalore-1:krn:vpc:123:subnet-456
```

## Behavior and Usage Notes

- All attributes require replacement if changed.
- Primary subnets cannot be deleted independently.
- If a subnet cannot be deleted because it is primary, Terraform will remove it from state.
- Update operations are not supported (all changes require recreation).