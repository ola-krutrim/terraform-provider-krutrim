---
page_title: "krutrim_kks_addons Data Source - terraform-provider-krutrim"
subcategory: "KKS"
description: |-
  Lists add-ons available for installation on KKS clusters.
---

# krutrim_kks_addons (Data Source)

Lists the add-ons that can be installed on KKS clusters in a region, along with their versions and the cluster versions they are compatible with.

Use it to find valid values for `addon_name` and `addon_version` of `krutrim_kks_addon`, and to see which Kubernetes cluster versions are supported.

## Example Usage

```hcl
data "krutrim_kks_addons" "catalog" {
  region = "In-Bangalore-1"
}

output "addon_names" {
  value = [for a in data.krutrim_kks_addons.catalog.addons : a.addon_name]
}

# Default version of the csi add-on for cluster version 1.34
locals {
  csi = one([for a in data.krutrim_kks_addons.catalog.addons : a if a.addon_name == "csi"])

  csi_default_version = one([
    for v in local.csi.addon_versions : v.addon_version
    if length([
      for c in v.compatibilities : c
      if c.cluster_version == "1.34" && c.default_version
    ]) > 0
  ])
}
```

## Schema

### Required

- `region` (String) Region to list add-ons for (e.g., `"In-Bangalore-1"`).

### Read-Only

- `addons` (Attributes List) Available add-ons. See [addons](#nested-schema-for-addons).

### Nested Schema for `addons`

Read-Only:

- `addon_name` (String) Add-on name, used as `addon_name` in `krutrim_kks_addon` (for example `csi`, `cilium`, `coredns`).
- `type` (String) Add-on category (for example `storage`, `networking`, `monitoring`).
- `description` (String) What the add-on does.
- `is_deletable` (Boolean) Whether the add-on can be removed after installation.
- `addon_versions` (Attributes List) Available versions. See [addon_versions](#nested-schema-for-addonsaddon_versions).

### Nested Schema for `addons.addon_versions`

Read-Only:

- `addon_version` (String) Version string, used as `addon_version` in `krutrim_kks_addon`.
- `architecture` (List of String) Supported CPU architectures (for example `amd64`, `arm64`).
- `compatibilities` (Attributes List) Cluster versions this add-on version works with. See [compatibilities](#nested-schema-for-addonsaddon_versionscompatibilities).

### Nested Schema for `addons.addon_versions.compatibilities`

Read-Only:

- `cluster_version` (String) Compatible Kubernetes cluster version.
- `platform_versions` (List of String) Compatible platform versions. `*` means all.
- `default_version` (Boolean) Whether this is the default version for that cluster version.

## Behavior and Usage Notes

- This data source is read-only. It never creates, modifies, or deletes anything.
- The distinct `cluster_version` values across all compatibilities show which Kubernetes versions you can use for `krutrim_kks_cluster`.
