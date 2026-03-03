---
page_title: "krutrim_security_group Resource - terraform-provider-krutrim"
subcategory: ""
description: |-
  Manages a Krutrim Security Group.
  This resource allows you to create and manage security groups inside a VPC.
  When applied, a new security group is created.
  When destroyed, the security group is deleted.
---

# krutrim_security_group (Resource)

Manages a Krutrim Security Group.

Security groups act as virtual firewalls that control inbound and outbound traffic for instances inside a VPC.

## Example Usage

```hcl
resource "krutrim_security_group" "example" {
  name        = "web-sg"
  description = "Security group for web servers"
  vpc_id      = "krn:vpc:123"
  region      = "In-Bangalore-1"
}
```

## Schema

### Required

- `name` (String) Name of the security group.
- `vpc_id` (String) VPC KRN where the security group belongs.
- `region` (String) Region (e.g., `"In-Bangalore-1"`).

### Optional

- `description` (String) Description of the security group.

### Computed

- `id` (String) Security Group ID (KRN).

## Behavior and Usage Notes

- Security group updates are not supported.
- Changing attributes requires recreation of the resource.
- If the security group is deleted outside Terraform, it will be removed from state during the next refresh.