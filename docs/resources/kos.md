# krutrim_kos_access_key (Resource)

Manages an Access Key for Krutrim Object Storage (KOS).

This resource creates an access key pair (access key + secret key) used to authenticate API requests to Krutrim Object Storage. The credentials are also used to create KOS sessions.

When applied:
- A new KOS access key and secret key are generated for the specified tier and region.

When destroyed:
- The access key is revoked and permanently deleted.

## Example Usage

```hcl
resource "krutrim_kos_access_key" "main" {
  region = "In-Bangalore-1"
  tier   = "tier-1"
}

output "access_key" {
  value = krutrim_kos_access_key.main.access_key
}

output "secret_key" {
  value     = krutrim_kos_access_key.main.secret_key
  sensitive = true
}
```

## Schema

### Required

- `region` (String) Region where the access key is created (e.g., `"In-Bangalore-1"`).
- `tier` (String) KOS storage tier (e.g., `"tier-1"`).

### Computed

- `id` (String) UUID of the access key resource.
- `access_key` (String) The generated KOS access key identifier.
- `secret_key` (String, Sensitive) The generated KOS secret key. Marked sensitive and will not appear in Terraform output.

## Behavior and Usage Notes

- The `secret_key` is only available at creation time. Store it securely (e.g., in Terraform state with a remote backend that supports encryption).
- This resource is a prerequisite for `krutrim_kos_session` and should be created first.
- Deleting and recreating this resource will generate new credentials — update any dependent sessions or configurations accordingly.
- `secret_key` is marked sensitive and will be redacted in Terraform plan and apply output.


# krutrim_kos_session (Resource)

Manages a KOS Session Token in Krutrim Cloud.

This resource creates a temporary session token by authenticating with a KOS access key and secret key. The session token is then used for authenticated operations such as uploading objects via `krutrim_kos_object`.

When applied:
- A session token is issued for the provided access key credentials.

When destroyed:
- The session token is invalidated.

## Example Usage

```hcl
resource "krutrim_kos_session" "main" {
  access_key = krutrim_kos_access_key.main.access_key
  secret_key = krutrim_kos_access_key.main.secret_key
  region     = "In-Bangalore-1"
  tier       = "tier-1"
}

output "session_token" {
  value     = krutrim_kos_session.main.session_token
  sensitive = true
}
```

## Schema

### Required

- `access_key` (String) The KOS access key. Typically sourced from `krutrim_kos_access_key`.
- `secret_key` (String, Sensitive) The KOS secret key. Typically sourced from `krutrim_kos_access_key`.
- `region` (String) Region where the session is created (e.g., `"In-Bangalore-1"`).
- `tier` (String) KOS storage tier (e.g., `"tier-1"`).

### Computed

- `id` (String) UUID of the session resource.
- `session_token` (String, Sensitive) The temporary session token for authenticating KOS operations.

## Behavior and Usage Notes

- The `session_token` is marked sensitive and will not appear in Terraform plan or apply output.
- Session tokens are temporary and may expire — if object operations fail, recreate the session resource.
- This resource depends on `krutrim_kos_access_key` and should be created after it.
- Use the `session_token` output as input to `krutrim_kos_object` for authenticated file uploads.


# krutrim_kos_bucket (Resource)

Manages a KOS Bucket in Krutrim Cloud.

A bucket is the fundamental container for objects in Krutrim Object Storage (KOS). This resource provisions a bucket with configurable versioning, anonymous access settings, and metadata tags.

When applied:
- A new KOS bucket is created in the specified region and tier.

When destroyed:
- The bucket is permanently deleted. The bucket must be empty before deletion.

## Example Usage

```hcl
resource "krutrim_kos_bucket" "main" {
  name   = "my-kos-bucket"
  region = "In-Bangalore-1"
  tier   = "tier-1"

  description      = "My KOS bucket"
  versioning       = true
  anonymous_access = false

  tags = {
    env     = "dev"
    project = "demo"
  }
}

output "bucket_id" {
  value = krutrim_kos_bucket.main.id
}
```

## Schema

### Required

- `name` (String) Globally unique name for the bucket. Must follow DNS naming conventions (lowercase letters, numbers, hyphens).
- `region` (String) Region where the bucket is created (e.g., `"In-Bangalore-1"`).
- `tier` (String) KOS storage tier (e.g., `"tier-1"`).

### Optional

- `description` (String) Human-readable description of the bucket.
- `versioning` (Boolean) Whether to enable object versioning. When enabled, previous versions of objects are retained on overwrite or delete. Defaults to `false`.
- `anonymous_access` (Boolean) Whether to allow unauthenticated public access to objects in the bucket. Defaults to `false`.
- `tags` (Map of String) Key-value metadata tags to associate with the bucket.

### Computed

- `id` (String) KRN (Krutrim Resource Name) of the bucket. Used as `bucket_krn` when creating `krutrim_kos_object` resources.

## Behavior and Usage Notes

- Bucket names must be globally unique across all KOS users and regions.
- Enabling `versioning` after initial creation is supported, but disabling it after enabling is not recommended as it may affect existing object versions.
- Set `anonymous_access = false` (default) to keep all objects private — objects can still be accessed via authenticated requests or pre-signed URLs.
- The `id` (KRN) of the bucket is used as the `bucket_krn` argument in `krutrim_kos_object`.
- The bucket must be empty before it can be destroyed. Remove all `krutrim_kos_object` resources first.


# krutrim_kos_object (Resource)

Manages a KOS Object in Krutrim Cloud.

This resource uploads a local file to a Krutrim Object Storage (KOS) bucket. The upload is authenticated using a session token obtained from `krutrim_kos_session`.

When applied:
- The specified local file is uploaded to the bucket under the given object key.

When destroyed:
- The object is permanently deleted from the bucket.

## Example Usage

```hcl
resource "krutrim_kos_object" "main" {
  bucket_krn    = krutrim_kos_bucket.main.id
  object_key    = "uploads/my-file.dat"
  region        = "In-Bangalore-1"
  session_token = krutrim_kos_session.main.session_token
  file_path     = "/path/to/local/file.dat"
}

output "download_url" {
  value = krutrim_kos_object.main.download_url
}
```

## Schema

### Required

- `bucket_krn` (String) KRN of the destination bucket. Use the `id` output of a `krutrim_kos_bucket` resource.
- `object_key` (String) The key (path) under which the object is stored in the bucket (e.g., `"uploads/file.dat"`).
- `region` (String) Region where the bucket resides (e.g., `"In-Bangalore-1"`).
- `session_token` (String, Sensitive) A valid KOS session token for authenticating the upload. Use the `session_token` output of a `krutrim_kos_session` resource.
- `file_path` (String) Absolute local path to the file to upload (e.g., `"/home/user/data/file.dat"`).

### Computed

- `id` (String) UUID of the object resource.
- `download_url` (String) A URL that can be used to download the uploaded object.

## Behavior and Usage Notes

- The `file_path` must be accessible from the machine running Terraform at apply time.
- Large file uploads are supported — the provider handles multipart upload internally.
- If the `session_token` expires before Terraform finishes the upload, the operation will fail. Recreate the `krutrim_kos_session` resource and re-apply.
- Changing `file_path` or `object_key` will replace the object resource.
- The `download_url` in the output may be scoped to authenticated access only if the bucket has `anonymous_access = false`.
- Ensure `krutrim_kos_bucket` and `krutrim_kos_session` are created before this resource.
