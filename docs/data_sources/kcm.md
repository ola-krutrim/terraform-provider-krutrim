---
page_title: "krutrim_kcm_cert Data Source - terraform-provider-krutrim"
subcategory: "KCM (Krutrim Certificate Manager)"
description: |-
  Retrieves details of an existing KCM Certificate in Krutrim Cloud.
---

# krutrim_kcm_cert (Data Source)

Provides information about an existing KCM Certificate.

This data source allows you to look up a certificate in the Krutrim Certificate Manager (KCM) by its name and VPC. Useful when referencing a certificate that was imported outside of the current Terraform configuration — for example, to attach it to a load balancer or other resource managed in a separate workspace.

## Example Usage

```hcl
data "krutrim_kcm_cert" "example" {
  name   = "my-tls-cert"
  vpc_id = "vpc-abc123"
}

output "cert_id" {
  value = data.krutrim_kcm_cert.example.id
}

output "cert_krn" {
  value = data.krutrim_kcm_cert.example.krn
}
```

## Schema

### Required

- `name` (String) The name of the certificate to look up. Must match the name used during import.
- `vpc_id` (String) The ID of the VPC the certificate is associated with.

### Read-Only

- `id` (String) UUID of the certificate resource.
- `krn` (String) KRN (Krutrim Resource Name) of the certificate.
- `expiration` (String) The expiration date of the certificate, as returned by KCM.
- `tags` (Map of String) Key-value metadata tags associated with the certificate.

## Behavior and Usage Notes

- If no certificate with the given `name` exists in the specified VPC, Terraform will return an error.
- This data source is read-only — it will never import, modify, or delete a certificate.
- Use this data source to reference certificates managed by a separate team or Terraform configuration without duplicating import logic.
- The `expiration` field reflects the validity period encoded in the certificate itself and is populated by KCM.
- Combine with other resources (e.g., load balancers) by passing `id` or `krn` as arguments where a certificate reference is required.
