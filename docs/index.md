---
page_title: "Krutrim Provider"
subcategory: ""
description: |-
  Terraform provider for managing infrastructure resources in Krutrim Cloud.
---

# Krutrim Provider

The **Krutrim Terraform Provider** allows you to provision and manage infrastructure resources in Krutrim Cloud.

Using this provider, you can manage:

- Virtual Private Clouds (VPCs)
- Subnets
- Virtual Machine Instances
- Floating IPs
- Security Groups and Security Group Rules
- SSH Keys
- Block Storage Volumes

---

## Example Usage

```hcl
terraform {
  required_providers {
    krutrim = {
      source  = "ola-krutrim/krutrim"
      version = "0.1.1"
    }
  }
}

provider "krutrim" {
  base_url = "https://cloud.olakrutrim.com"

  email        = "enter_email_here"
  password     = "enter_password_here"
  is_root_user = true
}
```

---

## Authentication

The provider authenticates using:

- `email`
- `password`
- `is_root_user`

### Provider Arguments

The following arguments are supported:

- `base_url` (String, Required)  
  Base URL of the Krutrim Cloud API.  
  Example: `"https://cloud.olakrutrim.com"`

- `email` (String, Required)  
  Login email for Krutrim Cloud.

- `password` (String, Required, Sensitive)  
  Account password.

- `is_root_user` (Boolean, Required)  
  Set to `true` if authenticating as root user.

---

## Environment Variable Support (Optional)

If supported in your provider implementation, credentials may also be supplied via environment variables:

```
KRUTRIM_BASE_URL
KRUTRIM_EMAIL
KRUTRIM_PASSWORD
KRUTRIM_IS_ROOT_USER
```

---

## Supported Resources

- `krutrim_vpc`
- `krutrim_subnet`
- `krutrim_instance`
- `krutrim_volume`
- `krutrim_floating_ip`
- `krutrim_security_group`
- `krutrim_security_group_rule`
- `krutrim_sshkey`

---

## Supported Data Sources

- `krutrim_vpc`
- `krutrim_subnet`
- `krutrim_instance`
- `krutrim_volume`
- `krutrim_floating_ip`
- `krutrim_sshkey`

---

## Notes

- Some resources (such as VPC and Instance) are created asynchronously. Terraform waits until provisioning completes.
- Deletions may fail if dependent resources still exist (e.g., volumes attached to instances).
- Many attributes use `RequiresReplace`, meaning changes will recreate the resource.