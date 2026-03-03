---
page_title: "krutrim_sshkey Resource - terraform-provider-krutrim"
subcategory: ""
description: |-
  Manages an SSH key in Krutrim Cloud.
  This resource allows you to create and manage SSH public keys
  used for authenticating to virtual machine instances.
---

# krutrim_sshkey (Resource)

Manages an SSH key in Krutrim Cloud.

This resource allows you to upload and manage SSH public keys that can be attached to VM instances for secure access.

When applied:
- A new SSH key is created in the specified region.

When destroyed:
- The SSH key is permanently deleted.

## Example Usage

```hcl
resource "krutrim_sshkey" "example" {
  key_name   = "my-ssh-key"
  public_key = file("~/.ssh/id_rsa.pub")
  region     = "In-Bangalore-1"
}
```

## Schema

### Required

- `key_name` (String) Name of the SSH key.
- `public_key` (String, Sensitive) SSH public key content.
- `region` (String) Region where the key is stored (e.g., `"In-Bangalore-1"`).

### Computed

- `id` (String) SSH key UUID.

## Behavior and Usage Notes

- The `public_key` value is marked as sensitive and will not be displayed in Terraform output.
- The resource uses UUID as the Terraform ID.
- If the SSH key is deleted outside Terraform, it will be removed from state during refresh.
- Updating the key content replaces the resource.