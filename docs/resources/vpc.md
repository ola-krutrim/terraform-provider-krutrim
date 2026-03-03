---
page_title: "krutrim_vpc Resource - terraform-provider-krutrim"
subcategory: ""
description: |-
  Manages a Krutrim Virtual Private Cloud (VPC).
  This resource creates a VPC along with its network and optional subnet.
---

# krutrim_vpc (Resource)

Manages a Krutrim Virtual Private Cloud (VPC).

This resource provisions:

- A VPC
- A network
- An optional subnet

Creation is asynchronous and Terraform waits for completion.

## Example Usage

```hcl
resource "krutrim_vpc" "example" {
  region       = "In-Bangalore-1"
  name         = "my-vpc"
  network_name = "my-network"

  subnet_name  = "my-subnet"
  cidr         = "10.0.0.0/24"
  gateway_ip   = "10.0.0.1"

  description  = "Production VPC"
  enabled      = true
  admin_state_up = true
  ip_version   = "4"
  ingress      = true
  egress       = true
}
```

## Schema

### Required

- `region` (String) Region where the VPC will be created.
- `name` (String) VPC name.
- `network_name` (String) Name of the network inside the VPC.

### Optional

- `subnet_name` (String) Subnet name.
- `cidr` (String) Subnet CIDR block.
- `gateway_ip` (String) Subnet gateway IP.
- `description` (String) VPC description.
- `subnet_description` (String) Subnet description.
- `enabled` (Boolean, default: true) Whether the VPC is enabled.
- `admin_state_up` (Boolean, default: true) Administrative state of the network.
- `ip_version` (String, default: "4") IP version (`4` or `6`).
- `ingress` (Boolean, default: true) Enable ingress on subnet.
- `egress` (Boolean, default: true) Enable egress on subnet.

### Computed

- `id` (String) VPC ID (KRN).
- `network_id` (String) Network ID (KRN).
- `subnet_id` (String) Subnet ID (KRN).

## Import

```
terraform import krutrim_vpc.example region:vpc_id
```

or

```
terraform import krutrim_vpc.example region:vpc_id:subnet_id
```

## Behavior and Usage Notes

- VPC creation is asynchronous and Terraform waits for completion.
- Subnet deletion is attempted before VPC deletion.
- If dependent resources (e.g., volumes) exist, deletion may fail.
- All changes require recreation.