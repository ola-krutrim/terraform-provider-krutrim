

# krutrim_dns_zone (Data Source)

Provides information about an existing DNS Zone.

This data source allows you to fetch DNS zone details by zone name, useful when creating DNS records inside a zone that was provisioned in a separate Terraform configuration or outside of Terraform entirely.

## Example Usage

```hcl
data "krutrim_dns_zone" "example" {
  zone_name = "example.internal"
}
```

## Schema

### Required

- `zone_name` (String) The domain name of the DNS zone to look up (e.g., `"example.internal"`).

### Read-Only

- `id` (String) KRN (Krutrim Resource Name) of the DNS zone. Use this as `krn_id` when creating `krutrim_dns_record` resources.
- `type` (String) The type of DNS zone (e.g., `"private"`).
- `vpc_id` (String) ID of the VPC associated with the DNS zone (for private zones).

## Behavior and Usage Notes

- If no DNS zone with the given `zone_name` exists, Terraform will return an error.
- This data source is read-only — it will never create, modify, or delete a DNS zone.
- The `id` (KRN) returned by this data source can be used directly as the `krn_id` argument in `krutrim_dns_record` resources.
- Useful for adding DNS records to a shared zone managed by another team or configuration.
---
page_title: "krutrim_dns_record Data Source - terraform-provider-krutrim"
subcategory: "DNS"
description: |-
  Retrieves details of an existing DNS Record within a DNS Zone in Krutrim Cloud.
---

# krutrim_dns_record (Data Source)

Provides information about an existing DNS Record.

This data source allows you to look up a specific DNS record by name and type within a given DNS zone. Useful when you need to reference or validate an existing record without managing it through Terraform.

## Example Usage

```hcl
data "krutrim_dns_record" "example" {
  krn_id = data.krutrim_dns_zone.example.id
  name   = "app"
  type   = "A"
}
```

## Schema

### Required

- `krn_id` (String) KRN of the parent DNS zone. Use the `id` output of a `krutrim_dns_zone` resource or data source.
- `name` (String) The subdomain label to look up (e.g., `"app"`, `"www"`).
- `type` (String) The DNS record type to look up. Accepted values: `"A"`, `"CNAME"`.

### Read-Only

- `id` (String) UUID of the DNS record.
- `ttl` (Number) Time-to-live in seconds configured for the record.
- `values` (List of String) The resolved values for this record (IP addresses for `A` records, target hostname for `CNAME` records).

## Behavior and Usage Notes

- If no record matching the given `name` and `type` exists within the specified zone, Terraform will return an error.
- This data source is read-only — it will never create, modify, or delete a DNS record.
- Multiple values in the `values` list indicate a round-robin DNS configuration.
- Combine with `data.krutrim_dns_zone` to look up both the zone and its records without hardcoding KRN IDs.
