---
page_title: "krutrim_kks_cluster Resource - terraform-provider-krutrim"
subcategory: "KKS"
description: |-
  Manages a Krutrim Kubernetes Service (KKS) cluster.
---

# krutrim_kks_cluster (Resource)

Manages a Krutrim Kubernetes Service (KKS) cluster.

This resource creates a managed Kubernetes control plane inside an existing VPC and subnet. Node groups and add-ons are managed separately with `krutrim_kks_node_group` and `krutrim_kks_addon`.

When applied:
- A new KKS cluster is created in the specified region.
- Terraform waits until the cluster status is `PROVISIONED` (up to 45 minutes).
- The cluster kubeconfig is fetched and stored in the `kubeconfig` attribute.

When destroyed:
- The cluster is deleted and Terraform waits until it is gone (up to 30 minutes).

## Example Usage

```hcl
resource "krutrim_kks_cluster" "example" {
  region                  = "In-Bangalore-1"
  name                    = "tf-kks"
  version                 = "1.34"
  vpc_krn                 = krutrim_vpc.example.id
  subnet_krns             = krutrim_subnet.example.id
  endpoint_private_access = false
  pod_ipv4_cidr           = "10.244.0.0/16"
  service_ipv4_cidr       = "10.96.0.0/12"
}

# Write the kubeconfig to a local file
resource "local_sensitive_file" "kubeconfig" {
  content  = krutrim_kks_cluster.example.kubeconfig
  filename = "${path.module}/kubeconfig.yaml"
}
```

## Schema

### Required

- `region` (String) Region where the cluster is created (e.g., `"In-Bangalore-1"`). Changing this forces a new resource.
- `name` (String) Name of the cluster. Changing this forces a new resource.
- `version` (String) Kubernetes version (for example `"1.34"`). Use the `krutrim_kks_addons` data source to see the cluster versions that are currently supported. Changing this forces a new resource.
- `vpc_krn` (String) KRN of the VPC the cluster runs in. Changing this forces a new resource.
- `subnet_krns` (String) Subnet KRN string as accepted by the KKS API (a single string, not a list). Changing this forces a new resource.
- `endpoint_private_access` (Boolean) Whether the Kubernetes API endpoint is reachable only from inside the VPC. Changing this forces a new resource.
- `pod_ipv4_cidr` (String) CIDR block used for Pod IP addresses (e.g., `"10.244.0.0/16"`). Changing this forces a new resource.
- `service_ipv4_cidr` (String) CIDR block used for Service IP addresses (e.g., `"10.96.0.0/12"`). Changing this forces a new resource.

### Computed

- `id` (String) Cluster KRN.
- `status` (String) Current cluster status (for example `PROVISIONED`).
- `version_supported` (Boolean) Whether the cluster's Kubernetes version is still supported.
- `created_at` (String) Unix creation timestamp as returned by KKS.
- `kubeconfig` (String, Sensitive) Cluster kubeconfig contents. Available after the cluster is `PROVISIONED`.

## Import

Import using the region and cluster KRN, separated by a colon:

```shell
terraform import krutrim_kks_cluster.example "In-Bangalore-1:<cluster_krn>"
```

## Behavior and Usage Notes

- Every argument is immutable. Changing any of them destroys and recreates the cluster.
- `kubeconfig` is sensitive and is stored in the Terraform state. Protect your state file accordingly.
- If the kubeconfig is not yet available right after provisioning, `kubeconfig` is left empty and is filled in by the next `terraform refresh` or `terraform apply`.
- Create waits up to 45 minutes and delete up to 30 minutes. If a run times out, check the Krutrim console before retrying, to avoid creating duplicates.
- If the cluster is deleted outside Terraform, it is removed from state during refresh.
- Delete node groups and add-ons before the cluster. Terraform does this automatically when they reference the cluster through `cluster_krn`.
