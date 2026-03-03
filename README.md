# Krutrim Terraform Provider

The **Krutrim Terraform Provider** allows you to manage infrastructure resources in Krutrim Cloud using Infrastructure as Code (IaC).

This provider enables provisioning and management of:

- Virtual Private Clouds (VPCs)
- Subnets
- Virtual Machine Instances
- Block Storage Volumes
- Floating IPs
- Security Groups
- Security Group Rules
- SSH Keys

---

## Requirements

- Terraform >= 1.3.0
- A valid Krutrim Cloud account

---

## Installation

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    krutrim = {
      source  = "ola-krutrim/krutrim"
      version = "0.1.2"
    }
  }
}
```

Then run:

```bash
terraform init
```

---

## Provider Configuration

```hcl
provider "krutrim" {
  base_url = "https://cloud.olakrutrim.com"

  email        = "your_email_here"
  password     = "your_password_here"
  is_root_user = true
}
```

---

## Provider Arguments

| Argument       | Type    | Required | Description |
|---------------|---------|----------|-------------|
| `base_url`     | String  | Yes      | Base URL of the Krutrim Cloud API |
| `email`        | String  | Yes      | Login email |
| `password`     | String  | Yes      | Account password |
| `is_root_user` | Boolean | Yes      | Set to `true` if authenticating as root user |

---

## Example: Create a VPC

```hcl
resource "krutrim_vpc" "example" {
  region       = "In-Bangalore-1"
  name         = "example-vpc"
  network_name = "example-network"

  subnet_name = "example-subnet"
  cidr        = "10.0.0.0/24"
  gateway_ip  = "10.0.0.1"
}
```

---

## Example: Create an Instance

```hcl
resource "krutrim_instance" "example" {
  region         = "In-Bangalore-1"
  name           = "example-instance"
  instance_type  = "standard.medium"

  vpc_id     = krutrim_vpc.example.id
  subnet_id  = krutrim_vpc.example.subnet_id
  network_id = krutrim_vpc.example.network_id

  volume_type = "ssd"
  volume_name = "root-volume"

  floating_ip = true
}
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

- Some resources are created asynchronously and Terraform waits for completion.
- Certain attributes require resource recreation if modified.
- Deletion may fail if dependent resources still exist.

---