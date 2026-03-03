---
page_title: "krutrim_instance Resource - terraform-provider-krutrim"
subcategory: ""
description: |-
  Manages a Krutrim Virtual Machine (VM) Instance.
  This resource allows you to provision and manage compute instances inside a VPC,
  attach volumes, assign floating IPs, and configure security groups.
  When applied, a new instance is created. When destroyed, the instance is deleted.
---

# krutrim_instance (Resource)

Manages a Krutrim Virtual Machine (VM) Instance.

This resource allows you to provision compute instances inside a VPC network with configurable:

- Instance type
- Volume configuration
- Floating IP assignment
- Security groups
- SSH key
- GPU option

When destroyed, the instance and optionally its volume are deleted (based on `delete_on_termination`).

## Example Usage

```hcl
resource "krutrim_instance" "example" {
  region         = "ap-south-1"
  name           = "my-vm"
  instance_type  = "standard.medium"

  vpc_id     = "krn:vpc:123"
  subnet_id  = "krn:subnet:123"
  network_id = "krn:network:123"

  volume_type = "ssd"
  volume_name = "my-root-volume"
  volume_size = 50

  image_krn   = "krn:image:ubuntu"
  sshkey_name = "my-ssh-key"

  is_gpu                = false
  floating_ip           = true
  delete_on_termination = true

  security_groups = [
    "krn:sg:001",
    "krn:sg:002"
  ]

  user_data = base64encode("#!/bin/bash\necho Hello")
}
```

## Schema

### Required

- `region` (String) Region where the instance will be created.
- `name` (String) Name of the instance.
- `instance_type` (String) Instance type (flavor).
- `vpc_id` (String) VPC KRN where the instance will be deployed.
- `subnet_id` (String) Subnet KRN.
- `network_id` (String) Network KRN.
- `volume_type` (String) Volume type (e.g., `ssd`, `hdd`).
- `volume_name` (String) Name of the root volume.

### Optional

- `image_krn` (String) Image KRN used to boot the instance.
- `sshkey_name` (String) SSH key name for login.
- `volume_size` (Number) Volume size in GB.
- `is_gpu` (Boolean, default: false) Enable GPU instance.
- `floating_ip` (Boolean, default: false) Allocate and attach a floating IP.
- `delete_on_termination` (Boolean, default: true) Delete attached volume when instance is destroyed.
- `security_groups` (List of String) Security Group KRNs.
- `user_data` (String, Sensitive) Base64 encoded cloud-init user data.

### Computed / Read-Only

- `id` (String) Instance KRN.
- `vm_name` (String) VM name returned by API.
- `status` (String) Current instance status (`BUILD`, `ACTIVE`, etc.).
- `private_ip_address` (String) Private IP address.
- `floating_ip_address` (String) Floating IP address (if enabled).
- `floating_ip_krn` (String) Floating IP KRN.
- `port_krn` (String) Network port KRN.
- `volume_krn` (String) Root volume KRN.

## Behavior and Usage Notes

- Instance creation waits until status becomes `ACTIVE`.
- If `floating_ip = true`, a floating IP is automatically created and attached.
- If `delete_on_termination = true`, root volume is deleted when instance is destroyed.
- Updating the instance is not supported — Terraform will recreate the resource.