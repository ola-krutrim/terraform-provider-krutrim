---
page_title: "krutrim_kks_node_group Resource - terraform-provider-krutrim"
subcategory: "KKS"
description: |-
  Manages a KKS node group within a cluster.
---

# krutrim_kks_node_group (Resource)

Manages a node group within a Krutrim Kubernetes Service (KKS) cluster.

A node group is a set of worker nodes that share the same instance flavor, disk size, and scaling settings.

When applied:
- A new node group is created in the cluster.
- Terraform waits until the node group is ready (up to 45 minutes).

When destroyed:
- The node group is deleted and Terraform waits until it is gone (up to 30 minutes).

## Example Usage

```hcl
resource "krutrim_kks_node_group" "default" {
  region         = "In-Bangalore-1"
  cluster_krn    = krutrim_kks_cluster.example.id
  name           = "default-pool"
  instance_types = "CPU-4x-16GB"
  disk_size      = 166
  subnets_krn    = krutrim_subnet.example.id

  scaling_config = {
    min_size     = 1
    max_size     = 5
    desired_size = 2
  }

  labels = {
    workload = "general"
  }

  taints = [
    {
      key    = "dedicated"
      value  = "batch"
      effect = "NoSchedule"
    }
  ]

  remote_access = {
    ssh_key_krn = krutrim_sshkey.example.id
  }

  node_repair_enabled = true
}
```

## Schema

### Required

- `region` (String) Region of the cluster (e.g., `"In-Bangalore-1"`). Changing this forces a new resource.
- `cluster_krn` (String) KRN of the cluster the node group belongs to. Changing this forces a new resource.
- `name` (String) Name of the node group. Changing this forces a new resource.
- `instance_types` (String) Instance flavor name (for example `"CPU-4x-16GB"`). Use the `krutrim_kks_flavors` data source to list valid values. Changing this forces a new resource.
- `disk_size` (Number) Disk size for each node. Changing this forces a new resource.
- `subnets_krn` (String) KRN of the subnet the nodes are placed in. Changing this forces a new resource.
- `scaling_config` (Attributes) Node count settings. This is the only argument that can be updated in place. See [scaling_config](#nested-schema-for-scaling_config).

### Optional

- `labels` (Map of String) Kubernetes labels applied to the nodes.
- `taints` (Attributes List) Kubernetes taints applied to the nodes. See [taints](#nested-schema-for-taints).
- `remote_access` (Attributes) SSH access settings for the nodes. See [remote_access](#nested-schema-for-remote_access).
- `node_repair_enabled` (Boolean) Whether automatic node repair is enabled. Changing this forces a new resource.

### Computed

- `id` (String) Node group KRN.
- `status` (String) Current node group status.
- `version` (String) Kubernetes version of the node group.
- `created_at` (String) Creation timestamp as returned by KKS.

### Nested Schema for `scaling_config`

Required:

- `min_size` (Number) Minimum number of nodes.
- `max_size` (Number) Maximum number of nodes.
- `desired_size` (Number) Desired number of nodes.

### Nested Schema for `taints`

Required:

- `key` (String) Taint key.
- `value` (String) Taint value.
- `effect` (String) Taint effect (for example `NoSchedule`, `PreferNoSchedule`, or `NoExecute`).

### Nested Schema for `remote_access`

Optional:

- `ssh_key_krn` (String) KRN of the SSH key used to access the nodes.
- `source_security_groups_krns` (List of String) KRNs of security groups allowed to reach the nodes over SSH.

## Import

Import using the region, cluster KRN, and node group KRN, separated by colons:

```shell
terraform import krutrim_kks_node_group.default "In-Bangalore-1:<cluster_krn>:<node_group_krn>"
```

## Behavior and Usage Notes

- Changing `scaling_config` updates the node group in place and Terraform waits for it to settle.
- `region`, `cluster_krn`, `name`, `instance_types`, `disk_size`, `subnets_krn`, and `node_repair_enabled` are immutable. Changing them destroys and recreates the node group.
- Only `scaling_config` is sent to the API on update. Changes to `labels`, `taints`, or `remote_access` after creation are not applied to the existing node group. To change them, recreate the node group, for example with `terraform apply -replace=krutrim_kks_node_group.default`.
- Create waits up to 45 minutes and delete up to 30 minutes.
- If the node group is deleted outside Terraform, it is removed from state during refresh.
