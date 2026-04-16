# krutrim_load_balancer (Resource)

Manages a Load Balancer in Krutrim Cloud.

This resource supports both **Application Load Balancers (ALB)** and **Network Load Balancers (NLB)**. It provisions a load balancer with one or more listeners that route traffic to backend target groups.

When applied:
- A load balancer is created in the specified VPC and subnet.
- Listeners are configured with routing rules and associated target groups.

When destroyed:
- The load balancer and its listeners are permanently deleted.

## Example Usage

### Application Load Balancer (ALB)

```hcl
resource "krutrim_load_balancer" "alb" {
  depends_on = [krutrim_target_group.tg_alb]

  region      = "In-Bangalore-1"
  lb_name     = "tf-alb"
  description = "Terraform ALB"

  create_port = true
  floating_ip = true

  vpc_id        = krutrim_vpc.vpc1.id
  network_id    = krutrim_vpc.vpc1.network_id
  vip_subnet_id = krutrim_vpc.vpc1.subnet_id

  lb_type = "ALB"
  flavor  = "standard"

  listeners = [
    {
      listener_name     = "http-listener"
      protocol          = "HTTP"
      protocol_port     = 80
      pool_name         = "alb-pool"
      lb_algorithm      = "ROUND_ROBIN"
      target_group_name = krutrim_target_group.tg_alb.name

      policy_name  = "alb-policy"
      action       = "REDIRECT_TO_POOL"
      compare_type = "EQUAL_TO"
      type         = "HOST_NAME"
      value        = "/"
    }
  ]
}
```

### Network Load Balancer (NLB)

```hcl
resource "krutrim_load_balancer" "nlb" {
  depends_on = [krutrim_target_group.tg]

  region      = "In-Bangalore-1"
  lb_name     = "tf-nlb"
  description = "Terraform NLB"

  create_port = true
  floating_ip = false

  vpc_id        = krutrim_vpc.vpc1.id
  network_id    = krutrim_vpc.vpc1.network_id
  vip_subnet_id = krutrim_vpc.vpc1.subnet_id

  lb_type = "NLB"
  flavor  = "standard"

  listeners = [
    {
      listener_name     = "tcp-listener"
      protocol          = "TCP"
      protocol_port     = 80
      pool_name         = "pool1"
      lb_algorithm      = "ROUND_ROBIN"
      target_group_name = krutrim_target_group.tg.name

      policy_name  = ""
      action       = ""
      compare_type = ""
      type         = ""
      value        = ""
    }
  ]
}
```

## Schema

### Required

- `region` (String) Region where the load balancer is created (e.g., `"In-Bangalore-1"`).
- `lb_name` (String) Name of the load balancer.
- `lb_type` (String) Type of load balancer. Accepted values: `"ALB"`, `"NLB"`.
- `flavor` (String) Load balancer flavor/size (e.g., `"standard"`).
- `vpc_id` (String) ID of the VPC in which to create the load balancer.
- `network_id` (String) ID of the network to attach the load balancer to.
- `vip_subnet_id` (String) ID of the subnet for the VIP (Virtual IP).
- `listeners` (List of Object) One or more listener configurations. See [Listener Schema](#listeners-schema) below.

### Optional

- `description` (String) Human-readable description of the load balancer.
- `create_port` (Boolean) Whether to automatically create a port for the VIP. Defaults to `true`.
- `floating_ip` (Boolean) Whether to associate a floating (public) IP with the load balancer. Defaults to `false`.

### Computed

- `id` (String) UUID of the load balancer.

---

### Listeners Schema

Each object in the `listeners` list supports the following fields:

#### Required

- `listener_name` (String) Name of the listener.
- `protocol` (String) Protocol for the listener. ALB supports `"HTTP"`, `"HTTPS"`; NLB supports `"TCP"`.
- `protocol_port` (Number) Port on which the listener accepts traffic.
- `pool_name` (String) Name of the backend pool created for this listener.
- `lb_algorithm` (String) Load balancing algorithm. Accepted values: `"ROUND_ROBIN"`, `"LEAST_CONNECTIONS"`, `"SOURCE_IP"`.
- `target_group_name` (String) Name of the target group to associate with this listener.

#### Required for ALB (set to `""` for NLB)

- `policy_name` (String) Name of the routing policy.
- `action` (String) Policy action (e.g., `"REDIRECT_TO_POOL"`).
- `compare_type` (String) Comparison type for routing rules (e.g., `"EQUAL_TO"`).
- `type` (String) Rule type (e.g., `"HOST_NAME"`, `"PATH"`).
- `value` (String) Value to match against the rule type.

---

## Behavior and Usage Notes

- `lb_type` determines the supported protocols: ALB supports HTTP/HTTPS routing; NLB supports raw TCP forwarding.
- Set `floating_ip = true` to expose the load balancer to public traffic via a floating IP.
- For NLB listeners, the policy fields (`policy_name`, `action`, `compare_type`, `type`, `value`) are required by the schema but should be set to empty strings (`""`).
- The resource depends on `krutrim_target_group` — always use `depends_on` or reference the target group name directly.
- Deleting this resource will remove all associated listeners and pools.
---
page_title: "krutrim_target_group Resource - terraform-provider-krutrim"
subcategory: "Load Balancer"
description: |-
  Manages a Target Group in Krutrim Cloud.
  A target group defines the backend members (VM instances) and health
  check configuration used by a load balancer listener.
---

# krutrim_target_group (Resource)

Manages a Target Group in Krutrim Cloud.

A target group holds one or more backend VM instances that receive traffic from a load balancer listener. It also defines how health checks are performed against those members.

When applied:
- A target group is created with the specified members and health monitor.

When destroyed:
- The target group is permanently deleted. Ensure no load balancer listener references it before destroying.

## Example Usage

```hcl
resource "krutrim_target_group" "tg_alb" {
  depends_on = [krutrim_instance.vm1]

  region = "In-Bangalore-1"
  vpc_id = krutrim_vpc.vpc1.id
  name   = "tg-alb"

  members = [
    {
      name          = krutrim_instance.vm1.name
      address       = krutrim_instance.vm1.private_ip_address
      protocol_port = 80
      weight        = 1
    }
  ]

  health_monitor = {
    type        = "HTTP"
    delay       = 30
    timeout     = 5
    max_retries = 3
    url_path    = "/"
  }

  lifecycle {
    ignore_changes = [members]
  }
}
```

## Schema

### Required

- `region` (String) Region where the target group is created (e.g., `"In-Bangalore-1"`).
- `vpc_id` (String) ID of the VPC in which the target group is created.
- `name` (String) Name of the target group.
- `members` (List of Object) List of backend member configurations. See [Members Schema](#members-schema) below.
- `health_monitor` (Object) Health check configuration. See [Health Monitor Schema](#health-monitor-schema) below.

### Computed

- `id` (String) UUID of the target group.

---

### Members Schema

Each object in the `members` list supports the following fields:

#### Required

- `name` (String) Display name for the member (typically the VM name).
- `address` (String) Private IP address of the backend instance.
- `protocol_port` (Number) Port on the backend instance that receives traffic.
- `weight` (Number) Relative weight for traffic distribution. Higher values receive proportionally more traffic.

---

### Health Monitor Schema

The `health_monitor` object supports the following fields:

#### Required

- `type` (String) Protocol used for health checks. Accepted values: `"HTTP"`, `"TCP"`.
- `delay` (Number) Interval in seconds between consecutive health checks.
- `timeout` (Number) Time in seconds to wait for a health check response before marking it as failed.
- `max_retries` (Number) Number of consecutive failures before marking a member as unhealthy.

#### Optional

- `url_path` (String) HTTP path to use for health checks. Only applicable when `type = "HTTP"`. Defaults to `"/"`.

---

## Behavior and Usage Notes

- Members can be added or removed outside of Terraform (e.g., via the console). Use `lifecycle { ignore_changes = [members] }` to prevent Terraform from overwriting those changes.
- The `health_monitor.type` should match the protocol of the load balancer listener (e.g., use `"HTTP"` for ALB HTTP listeners).
- A target group must exist before creating a load balancer that references it — use `depends_on` accordingly.
- Destroying this resource while a load balancer listener still references it will result in an error.
