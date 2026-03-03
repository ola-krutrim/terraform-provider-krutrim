---
page_title: "krutrim_sshkey Data Source - terraform-provider-krutrim"
subcategory: ""
description: |-
  Retrieves details of an existing SSH key in Krutrim Cloud.
---

# krutrim_sshkey (Data Source)

Provides information about an existing SSH key.

This data source allows you to fetch SSH key details by name.

## Example Usage

```hcl
data "krutrim_sshkey" "example" {
  key_name = "my-ssh-key"
  region   = "In-Bangalore-1"
}
```

## Schema

### Required

- `key_name` (String) Name of the SSH key.
- `region` (String) Region where the SSH key exists.

### Read-Only

- `id` (String) SSH key UUID.
- `public_key` (String, Sensitive) SSH public key content.
- `region` (String) Region where the key is stored.

## Behavior and Usage Notes

- If the SSH key does not exist, Terraform will return an error.
- The `public_key` attribute is marked sensitive.
- Useful when referencing an existing SSH key for VM creation.