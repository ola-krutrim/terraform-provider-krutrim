---
page_title: "krutrim_iam_programmatic_access Data Source - terraform-provider-krutrim"
subcategory: "IAM"
description: |-
  Retrieves details of an existing programmatic access configuration for a Krutrim IAM user.
---

# krutrim_iam_programmatic_access (Data Source)

Retrieves the programmatic access details for an existing IAM user. Use this data source when the access key was created outside of the current Terraform configuration and you need to reference it.

## Example Usage

```hcl
data "krutrim_iam_programmatic_access" "example" {
  user_krn = "kcs::global::1234567890::abcd-efgh::iam::users::my-user"
}
```

### Reference Alongside a User Data Source

```hcl
data "krutrim_iam_user" "existing" {
  user_name = "john-doe"
}

data "krutrim_iam_programmatic_access" "example" {
  user_krn = data.krutrim_iam_user.existing.user_krn
}

output "access_key" {
  description = "Access key for the existing user"
  value       = data.krutrim_iam_programmatic_access.example.access_key
}
```

## Schema

### Required

- `user_krn` (String) KRN of the IAM user whose programmatic access details you want to look up.

### Read-Only

- `id` (String) Unique identifier of the programmatic access record.
- `access_key` (String) The access key ID associated with the user.
- `secret_key` (String, Sensitive) The secret access key. **Note: this field will be empty if the secret was not stored in Terraform state at creation time — the API does not expose the secret after initial creation.**

## Behavior and Usage Notes

- This data source is read-only — it will never create, modify, or delete programmatic access credentials.
- If no programmatic access exists for the given `user_krn`, Terraform will return an error.
- The `secret_key` is only available in the Terraform state if the resource was originally created using `krutrim_iam_programmatic_access` in the same or a remote state. If credentials were created outside Terraform, the `secret_key` field will be empty.
- Use this data source to pass the `access_key` to other resources or modules that need API credentials without re-creating them.
- Combine with `data.krutrim_iam_user` to look up both user details and their programmatic access in a single configuration.