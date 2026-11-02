---
page_title: "krutrim_kks_flavors Data Source - terraform-provider-krutrim"
subcategory: "KKS"
description: |-
  Lists instance flavors available for KKS node groups.
---

# krutrim_kks_flavors (Data Source)

Lists the instance flavors available for KKS node groups in a region.

Use it to find valid values for the `instance_types` argument of `krutrim_kks_node_group`.

## Example Usage

```hcl
data "krutrim_kks_flavors" "available" {
  region = "In-Bangalore-1"
}

output "flavor_names" {
  value = [for f in data.krutrim_kks_flavors.available.flavors : f.name]
}

# Pick a flavor by name
locals {
  gpu_flavors = [
    for f in data.krutrim_kks_flavors.available.flavors : f.name if f.gpu_count > 0
  ]
}
```

## Schema

### Required

- `region` (String) Region to list flavors for (e.g., `"In-Bangalore-1"`).

### Read-Only

- `flavors` (Attributes List) Available flavors. See [flavors](#nested-schema-for-flavors).

### Nested Schema for `flavors`

Read-Only:

- `name` (String) Flavor name, used as `instance_types` in `krutrim_kks_node_group` (for example `CPU-4x-16GB`).
- `cpu` (Number) Number of vCPUs.
- `memory` (String) Amount of memory.
- `gpu_count` (Number) Number of GPUs. `0` for CPU-only flavors.
- `gpu_memory` (Number) GPU memory. `0` for CPU-only flavors.

## Behavior and Usage Notes

- This data source is read-only. It never creates, modifies, or deletes anything.
- The available flavors can differ between regions and accounts.
