---
page_title: "krutrim_floating_ip Data Source - terraform-provider-krutrim"
subcategory: ""
description: |-
  Retrieves details of an existing Floating IP resource.
---

# krutrim_floating_ip (Data Source)

Provides information about an existing Floating IP.

## Example Usage

```hcl
data "krutrim_floating_ip" "example" {
  id     = "krn:floatingip:123"
  region = "ap-south-1"
  vpc_id = "krn:vpc:123"
}
```

## Schema

### Required

- `id` (String) Floating IP KRN.
- `region` (String) Region.
- `vpc_id` (String) VPC KRN.

### Read-Only

- `id` (String)