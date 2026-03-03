---
page_title: "krutrim_vpc Data Source - terraform-provider-krutrim"
subcategory: ""
description: |-
  Retrieves details of an existing Krutrim VPC.
---

# krutrim_vpc (Data Source)

Provides information about an existing VPC.

## Example Usage

```hcl
data "krutrim_vpc" "example" {
  id     = "krn:vpc:123"
  region = "In-Bangalore-1"
}
```

## Schema

### Required

- `id` (String) VPC ID (KRN).
- `region` (String) Region.

### Read-Only

- `name` (String) VPC name.
- `network_id` (String) Network ID.
- `subnet_id` (String) Subnet ID.