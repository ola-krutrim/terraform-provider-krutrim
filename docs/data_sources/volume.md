---
page_title: "krutrim_volume Data Source - terraform-provider-krutrim"
subcategory: ""
description: |-
  Retrieves details of an existing Krutrim Block Storage Volume.
---

# krutrim_volume (Data Source)

Provides information about an existing volume.

## Example Usage

```hcl
data "krutrim_volume" "example" {
  id          = "volume-123"
  k_tenant_id = "tenant-123"
}
```

## Schema

### Required

- `id` (String) Volume ID.
- `k_tenant_id` (String) Tenant ID.

### Read-Only

- `name` (String) Volume name.
- `size` (Number) Volume size in GB.
- `description` (String) Volume description.
- `volume_type` (String) Volume type.
- `multiattach` (Boolean) Multiattach enabled.