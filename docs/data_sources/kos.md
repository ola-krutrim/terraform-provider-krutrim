---
page_title: "krutrim_kos_access_key Data Source - terraform-provider-krutrim"
subcategory: "KOS (Krutrim Object Storage)"
description: |-
  Retrieves details of an existing KOS Access Key in Krutrim Cloud.
---

# krutrim_kos_access_key (Data Source)

Provides information about an existing KOS Access Key.

This data source allows you to fetch access key metadata by region and tier. Useful when referencing credentials generated in a separate Terraform workspace or created outside of Terraform.

## Example Usage

```hcl
data "krutrim_kos_access_key" "example" {
  region = "In-Bangalore-1"
  tier   = "tier-1"
}
```

## Schema

### Required

- `region` (String) Region where the access key was created (e.g., `"In-Bangalore-1"`).
- `tier` (String) KOS storage tier the access key belongs to (e.g., `"tier-1"`).

### Read-Only

- `id` (String) UUID of the access key resource.
- `access_key` (String) The KOS access key identifier.
- `secret_key` (String, Sensitive) The KOS secret key. Marked sensitive and will not appear in Terraform output.

## Behavior and Usage Notes

- If no access key exists for the given region and tier, Terraform will return an error.
- This data source is read-only — it will never create, modify, or delete an access key.
- The `secret_key` attribute is marked sensitive and will be redacted in Terraform plan and apply output.
- Use this data source to pass credentials into a `krutrim_kos_session` without duplicating credential management across configurations.
---
page_title: "krutrim_kos_session Data Source - terraform-provider-krutrim"
subcategory: "KOS (Krutrim Object Storage)"
description: |-
  Retrieves details of an existing KOS Session in Krutrim Cloud.
---

# krutrim_kos_session (Data Source)

Provides information about an existing KOS Session.

This data source allows you to look up a KOS session by access key, region, and tier. Useful when a session token was generated in a separate configuration and needs to be referenced for object operations.

## Example Usage

```hcl
data "krutrim_kos_session" "example" {
  access_key = data.krutrim_kos_access_key.example.access_key
  region     = "In-Bangalore-1"
  tier       = "tier-1"
}
```

## Schema

### Required

- `access_key` (String) The KOS access key associated with the session.
- `region` (String) Region where the session was created (e.g., `"In-Bangalore-1"`).
- `tier` (String) KOS storage tier associated with the session (e.g., `"tier-1"`).

### Read-Only

- `id` (String) UUID of the session resource.
- `session_token` (String, Sensitive) The temporary session token for authenticating KOS operations. Marked sensitive and will not appear in Terraform output.

## Behavior and Usage Notes

- If no active session exists for the given `access_key`, region, and tier, Terraform will return an error.
- This data source is read-only — it will never create, modify, or invalidate a session.
- The `session_token` is marked sensitive and will be redacted in Terraform plan and apply output.
- Session tokens are temporary and may expire. If dependent object operations fail, recreate the session using the `krutrim_kos_session` resource instead.


---
page_title: "krutrim_kos_bucket Data Source - terraform-provider-krutrim"
subcategory: "KOS (Krutrim Object Storage)"
description: |-
  Retrieves details of an existing KOS Bucket in Krutrim Cloud.
---

# krutrim_kos_bucket (Data Source)

Provides information about an existing KOS Bucket.

This data source allows you to fetch bucket details by name and region. Useful when referencing a bucket created outside of the current Terraform configuration, for example to upload objects into a shared bucket.

## Example Usage

```hcl
data "krutrim_kos_bucket" "example" {
  name   = "my-kos-bucket"
  region = "In-Bangalore-1"
}
```

## Schema

### Required

- `name` (String) Name of the KOS bucket to look up.
- `region` (String) Region where the bucket exists (e.g., `"In-Bangalore-1"`).

### Read-Only

- `id` (String) KRN (Krutrim Resource Name) of the bucket. Use this as `bucket_krn` when creating `krutrim_kos_object` resources.
- `tier` (String) Storage tier of the bucket (e.g., `"tier-1"`).
- `description` (String) Human-readable description of the bucket.
- `versioning` (Boolean) Whether object versioning is enabled on the bucket.
- `anonymous_access` (Boolean) Whether the bucket allows unauthenticated public access.
- `tags` (Map of String) Key-value metadata tags associated with the bucket.

## Behavior and Usage Notes

- If no bucket with the given `name` exists in the specified region, Terraform will return an error.
- This data source is read-only — it will never create, modify, or delete a bucket.
- The `id` (KRN) returned by this data source can be used directly as the `bucket_krn` argument in `krutrim_kos_object` resources.
- Useful when the bucket is managed by a separate team or configuration and you only need to upload or reference objects within it.
---
page_title: "krutrim_kos_object Data Source - terraform-provider-krutrim"
subcategory: "KOS (Krutrim Object Storage)"
description: |-
  Retrieves details of an existing Object within a KOS Bucket in Krutrim Cloud.
---

# krutrim_kos_object (Data Source)

Provides information about an existing KOS Object.

This data source allows you to look up a specific object within a KOS bucket by its key. Useful when referencing an object's metadata or download URL without managing the object's lifecycle through Terraform.

## Example Usage

```hcl
data "krutrim_kos_object" "example" {
  bucket_krn    = data.krutrim_kos_bucket.example.id
  object_key    = "uploads/my-file.dat"
  region        = "In-Bangalore-1"
  session_token = krutrim_kos_session.main.session_token
}
```

## Schema

### Required

- `bucket_krn` (String) KRN of the bucket where the object resides. Use the `id` output of a `krutrim_kos_bucket` resource or data source.
- `object_key` (String) The key (path) of the object within the bucket (e.g., `"uploads/my-file.dat"`).
- `region` (String) Region where the bucket exists (e.g., `"In-Bangalore-1"`).
- `session_token` (String, Sensitive) A valid KOS session token for authenticating the lookup request.

### Read-Only

- `id` (String) UUID of the object resource.
- `download_url` (String) URL to download the object. Access may require authentication depending on bucket settings.

## Behavior and Usage Notes

- If no object with the given `object_key` exists in the specified bucket, Terraform will return an error.
- This data source is read-only — it will never upload, modify, or delete an object.
- The `session_token` is required for authenticated access even for read operations.
- The `download_url` may be publicly accessible or require credentials depending on the bucket's `anonymous_access` setting.
- Combine with `data.krutrim_kos_bucket` to avoid hardcoding bucket KRN values across configurations.
