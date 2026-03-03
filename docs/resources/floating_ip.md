---
page_title: "krutrim_floating_ip Resource - terraform-provider-krutrim"
subcategory: ""
description: |-
  Manages a Floating IP resource in Krutrim Cloud.
  This resource allows you to allocate and release floating IP addresses inside a VPC.
---

# krutrim_floating_ip (Resource)

Manages a Floating IP resource.

This resource allocates a floating IP inside a VPC and makes it available for attachment.

## Example Usage

```hcl
resource "krutrim_floating_ip" "example" {
  region = "ap-south-1"
  name   = "my-floating-ip"
  vpc_id = "krn:vpc:123"
}
```

## Schema

### Required

- `region` (String) Region of the VPC.
- `name` (String) Name of the floating IP.
- `vpc_id` (String) VPC KRN.

### Optional

- `floating_ip` (Boolean, default: true) Allocate public floating IP.

### Computed

- `id` (String) Floating IP KRN.

## Import

Floating IP can be imported using:

```bash
terraform import krutrim_floating_ip.example region:floating_ip_krn
```

Example:

```bash
terraform import krutrim_floating_ip.example ap-south-1:krn:fip:12345
```

## Behavior Notes

- Floating IP is resolved using VPC network and active subnet.
- Deletion waits until the floating IP is fully released.