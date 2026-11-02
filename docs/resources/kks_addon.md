---
page_title: "krutrim_kks_addon Resource - terraform-provider-krutrim"
subcategory: "KKS"
description: |-
  Manages a KKS add-on installed on a cluster.
---

# krutrim_kks_addon (Resource)

Manages an add-on installed on a Krutrim Kubernetes Service (KKS) cluster.

Use the `krutrim_kks_addons` data source to see which add-ons and versions are available for your cluster version.

When applied:
- The add-on is installed on the cluster.
- Terraform waits until the add-on is ready (up to 45 minutes).

When destroyed:
- The add-on is removed from the cluster and Terraform waits until it is gone (up to 30 minutes).

## Example Usage

```hcl
resource "krutrim_kks_addon" "csi" {
  region        = "In-Bangalore-1"
  cluster_krn   = krutrim_kks_cluster.example.id
  addon_name    = "csi"
  addon_version = "1.0.0"

  # Many add-ons need worker nodes to run on
  depends_on = [krutrim_kks_node_group.default]
}

# Install the server default version
resource "krutrim_kks_addon" "metrics_server" {
  region      = "In-Bangalore-1"
  cluster_krn = krutrim_kks_cluster.example.id
  addon_name  = "metrics-server"

  depends_on = [krutrim_kks_node_group.default]
}
```

## Schema

### Required

- `region` (String) Region of the cluster (e.g., `"In-Bangalore-1"`). Changing this forces a new resource.
- `cluster_krn` (String) KRN of the cluster to install the add-on on. Changing this forces a new resource.
- `addon_name` (String) Name of the add-on (for example `"csi"`, `"cilium"`, `"coredns"`). Changing this forces a new resource.

### Optional

- `addon_version` (String) Version of the add-on. Omit to use the server default version. Changing this forces a new resource.

### Computed

- `id` (String) Installed add-on KRN.
- `status` (String) Current add-on status.
- `cluster_name` (String) Name of the cluster the add-on is installed on.
- `created_at` (String) Creation timestamp as returned by KKS.

## Import

Import using the region, cluster KRN, and installed add-on KRN, separated by colons:

```shell
terraform import krutrim_kks_addon.csi "In-Bangalore-1:<cluster_krn>:<addon_krn>"
```

## Behavior and Usage Notes

- All arguments are immutable. Changing any of them destroys and recreates the add-on.
- Add-ons that run workloads, such as `csi`, need nodes to be scheduled on. Add a `depends_on` for your node group so the nodes exist first.
- When `addon_version` is omitted, the version chosen by the server is stored in state.
- Create waits up to 45 minutes and delete up to 30 minutes.
- If the add-on is deleted outside Terraform, it is removed from state during refresh.
