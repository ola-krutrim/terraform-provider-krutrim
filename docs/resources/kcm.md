---
page_title: "krutrim_kcm_cert Resource - terraform-provider-krutrim"
subcategory: "KCM (Krutrim Certificate Manager)"
description: |-
  Manages a KCM Certificate in Krutrim Cloud.
---

# krutrim_kcm_cert (Resource)

Manages a KCM Certificate in Krutrim Cloud.

This resource imports and manages an SSL/TLS certificate in the Krutrim Certificate Manager (KCM). The certificate is uploaded from a local file and associated with a VPC. Optional metadata tags can be attached for organization and filtering.

When applied:
- The specified certificate file is imported into KCM and associated with the given VPC.

When destroyed:
- The certificate is permanently deleted from KCM.

## Example Usage

```hcl
resource "krutrim_kcm_cert" "main" {
  name      = "my-tls-cert"
  vpc_id    = "vpc-abc123"
  file_path = "/path/to/certificate.pem"

  tags = {
    env     = "production"
    project = "web-app"
  }
}

output "cert_id" {
  value = krutrim_kcm_cert.main.id
}

output "cert_krn" {
  value = krutrim_kcm_cert.main.krn
}

output "cert_expiration" {
  value = krutrim_kcm_cert.main.expiration
}
```

## Schema

### Required

- `name` (String) A unique name for the certificate within the VPC. Used to identify the certificate after import.
- `vpc_id` (String) The ID of the VPC to associate the certificate with.
- `file_path` (String) Absolute local path to the certificate file to upload (e.g., `"/home/user/certs/cert.pem"`). The file must be accessible from the machine running Terraform at apply time.

### Optional

- `tags` (Map of String) Key-value metadata tags to associate with the certificate.

### Computed

- `id` (String) UUID of the certificate resource.
- `krn` (String) KRN (Krutrim Resource Name) of the certificate.
- `expiration` (String) The expiration date of the certificate, as returned by KCM.

## Behavior and Usage Notes

- The `file_path` must point to a valid, accessible certificate file at apply time.
- After import, the provider retries the certificate list API up to 5 times (with 2-second intervals) to confirm the certificate is available before saving state.
- If the certificate cannot be found after all retries, Terraform will return an error — this may indicate the import failed silently or the API is temporarily unavailable.
- Tags are applied after the certificate is imported. A failure in tag assignment does not block the resource from being created.
- Changing `name`, `vpc_id`, or `file_path` may trigger a replacement or update depending on which field changes.
- Updating the certificate (`file_path` change) re-uploads the certificate file and refreshes `krn` and `expiration` from the API.
- The `expiration` field reflects the validity period encoded in the certificate itself and is populated by KCM on import.
