---
page_title: "krutrim_iam_programmatic_access Resource - terraform-provider-krutrim"
subcategory: "IAM"
description: |-
  Manages programmatic access (access key + secret key) for a Krutrim IAM user.
---

# krutrim_iam_programmatic_access (Resource)

Enables programmatic access for an IAM user by generating an access key and secret key pair. Use this resource when you need API-level access to Krutrim Cloud services without console login.

## Example Usage

```hcl
resource "krutrim_iam_programmatic_access" "this" {
  user_krn = krutrim_iam_user.this.user_krn

  depends_on = [krutrim_iam_user.this]
}
```

### Full Example — User, Role Binding, and Programmatic Access

```hcl
# 1. Create IAM User
resource "krutrim_iam_user" "this" {
  user_name      = "john-doe"
  email          = "john.doe@example.com"
  password       = "YourStr0ngPass!"
  console_access = true

  lifecycle {
    ignore_changes = [password]
  }
}

# 2. Bind an existing role to the user
resource "krutrim_iam_user_role_binding" "this" {
  user_id = krutrim_iam_user.this.id

  role_ids = [
    "kcs::global::0000000000::00000000-0000-0000-0000-000000000000::iam::roles::3e79b4c8-6a86-497e-8d2b-3203b55faafb"
  ]

  depends_on = [krutrim_iam_user.this]
}

# 3. Enable Programmatic Access
resource "krutrim_iam_programmatic_access" "this" {
  user_krn = krutrim_iam_user.this.user_krn

  depends_on = [krutrim_iam_user.this]
}

# Outputs
output "user_id" {
  description = "ID of the created IAM user"
  value       = krutrim_iam_user.this.id
}

output "user_krn" {
  description = "KRN of the created IAM user"
  value       = krutrim_iam_user.this.user_krn
}

output "access_key" {
  description = "Programmatic access key"
  value       = krutrim_iam_programmatic_access.this.access_key
}

output "secret_key" {
  description = "Programmatic secret key — save immediately, shown only once"
  value       = krutrim_iam_programmatic_access.this.secret_key
  sensitive   = true
}
```

## Schema

### Required

- `user_krn` (String) KRN of the IAM user for whom programmatic access is being enabled. Use the `user_krn` attribute of a `krutrim_iam_user` resource.

### Read-Only

- `id` (String) Unique identifier of the programmatic access record.
- `access_key` (String) The access key ID used for API authentication.
- `secret_key` (String, Sensitive) The secret access key used alongside the access key. **This value is only available at creation time and cannot be retrieved again. Save it immediately.**

## Behavior and Usage Notes

- The `secret_key` is shown only once — immediately after `terraform apply`. It is not retrievable from the API afterwards. Store it securely in a secrets manager before the Terraform state is refreshed or shared.
- Always mark the `secret_key` output as `sensitive = true` to prevent it from appearing in plaintext in logs or terminal output.
- The `depends_on` argument is recommended to ensure the IAM user exists before programmatic access is provisioned.
- Destroying this resource will revoke the access key and secret key pair for the user.
- Only one programmatic access resource can be active per user at a time. Creating a new one will replace the existing credentials.