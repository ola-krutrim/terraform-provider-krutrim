# krutrim_dns_zone (Resource)

Manages a DNS Zone in Krutrim Cloud.

A DNS zone is a container for DNS records associated with a domain. In Krutrim Cloud, DNS zones can be scoped privately to a VPC, enabling internal name resolution for resources within that VPC.

When applied:
- A DNS zone is created with the specified name and type.

When destroyed:
- The DNS zone and all its associated records are permanently deleted.

## Example Usage

```hcl
resource "krutrim_dns_zone" "example" {
  type      = "private"
  vpc_id    = krutrim_vpc.vpc1.id
  zone_name = "example.internal"
}
```

## Schema

### Required

- `zone_name` (String) The domain name for the DNS zone (e.g., `"example.internal"`, `"test.com"`).
- `type` (String) The type of DNS zone. Accepted values: `"private"`.

### Optional

- `vpc_id` (String) ID of the VPC to associate with a private DNS zone. Required when `type = "private"`.

### Computed

- `id` (String) KRN (Krutrim Resource Name) of the DNS zone. Used as `krn_id` when creating DNS records.

## Behavior and Usage Notes

- Private DNS zones are only resolvable within the associated VPC.
- The `vpc_id` is required for `type = "private"` zones.
- The `id` output is the KRN identifier used when creating `krutrim_dns_record` resources.
- Deleting a zone will also delete all DNS records within it — ensure dependent `krutrim_dns_record` resources are destroyed first, or they will be removed automatically.
- Zone names should follow standard DNS naming conventions (e.g., `"myapp.internal"`).


# krutrim_dns_record (Resource)

Manages a DNS Record in Krutrim Cloud.

This resource creates DNS records (such as A, CNAME) within an existing `krutrim_dns_zone`. Multiple values can be specified for round-robin or failover configurations.

When applied:
- A DNS record is created within the referenced DNS zone.

When destroyed:
- The DNS record is permanently deleted from the zone.

## Example Usage

### Single A Record

```hcl
resource "krutrim_dns_record" "record_basic" {
  krn_id = krutrim_dns_zone.example.id

  name   = "app"
  type   = "A"
  ttl    = 300
  values = ["10.0.0.10"]
}
```

### Multiple A Records (Round-Robin)

```hcl
resource "krutrim_dns_record" "record_multi" {
  krn_id = krutrim_dns_zone.example.id

  name   = "api"
  type   = "A"
  ttl    = 300
  values = [
    "10.0.0.11",
    "10.0.0.12"
  ]
}
```

### CNAME Record

```hcl
resource "krutrim_dns_record" "record_cname" {
  krn_id = krutrim_dns_zone.example.id

  name   = "www"
  type   = "CNAME"
  ttl    = 300
  values = ["app.example.internal"]
}
```

## Schema

### Required

- `krn_id` (String) KRN of the parent DNS zone. Use the `id` output of a `krutrim_dns_zone` resource.
- `name` (String) The subdomain name for this record (e.g., `"app"`, `"www"`). This is relative to the zone name.
- `type` (String) The DNS record type. Accepted values: `"A"`, `"CNAME"`.
- `ttl` (Number) Time-to-live in seconds for the DNS record. Common values: `300` (5 min), `3600` (1 hr).
- `values` (List of String) One or more values for the DNS record. For `A` records, provide IP addresses. For `CNAME` records, provide the target hostname.

### Computed

- `id` (String) UUID of the DNS record.

## Behavior and Usage Notes

- Multiple entries in `values` for an `A` record result in round-robin DNS resolution.
- `CNAME` records should have exactly one value pointing to the target hostname.
- The `name` field is a relative label within the zone — the full FQDN will be `<name>.<zone_name>`.
- Updating `values` or `ttl` will update the record in-place without replacement.
- The record depends on an existing `krutrim_dns_zone` — always reference the zone's `id` via `krn_id`.
