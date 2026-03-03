---
page_title: "krutrim_instance Data Source - terraform-provider-krutrim"
subcategory: ""
description: |-
  Retrieves details of an existing Krutrim Virtual Machine Instance.
---

# krutrim_instance (Data Source)

Provides information about an existing VM instance.

## Example Usage

```hcl
data "krutrim_instance" "example" {
  id     = "krn:instance:123"
  region = "ap-south-1"
}
```

## Schema

### Required

- `id` (String) Instance KRN.
- `region` (String) Region of the instance.

### Read-Only

- `vm_name` (String)
- `status` (String)
- `private_ip_address` (String)
- `floating_ip_address` (String)
- `volume_krn` (String)
- `port_krn` (String)
- `floating_ip_krn` (String)