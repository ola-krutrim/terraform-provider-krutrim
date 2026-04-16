---
page_title: "krutrim_load_balancer Data Source - terraform-provider-krutrim"
subcategory: "Load Balancer"
description: |-
  Retrieves details of an existing Load Balancer in Krutrim Cloud.
---

# krutrim_load_balancer (Data Source)

Provides information about an existing Load Balancer (ALB or NLB).

This data source allows you to fetch load balancer details by name and region, useful when referencing a load balancer created outside of the current Terraform configuration.

## Example Usage

```hcl
data "krutrim_load_balancer" "example" {
  lb_name = "tf-alb"
  region  = "In-Bangalore-1"
}
```

## Schema

### Required

- `lb_name` (String) Name of the load balancer to look up.
- `region` (String) Region where the load balancer exists (e.g., `"In-Bangalore-1"`).

### Read-Only

- `id` (String) UUID of the load balancer.
- `lb_type` (String) Type of the load balancer — `"ALB"` or `"NLB"`.
- `flavor` (String) Flavor/size of the load balancer (e.g., `"standard"`).
- `description` (String) Human-readable description of the load balancer.
- `vpc_id` (String) ID of the VPC the load balancer is associated with.
- `network_id` (String) ID of the network attached to the load balancer.
- `vip_subnet_id` (String) ID of the subnet used for the VIP (Virtual IP).
- `floating_ip` (Boolean) Whether a floating (public) IP is associated with the load balancer.
- `listeners` (List of Object) List of listener configurations attached to the load balancer. See [Listeners Schema](#listeners-schema) below.

---

### Listeners Schema

Each object in the `listeners` list exposes the following read-only fields:

- `listener_name` (String) Name of the listener.
- `protocol` (String) Protocol of the listener (`"HTTP"`, `"HTTPS"`, or `"TCP"`).
- `protocol_port` (Number) Port on which the listener accepts traffic.
- `pool_name` (String) Name of the backend pool associated with this listener.
- `lb_algorithm` (String) Load balancing algorithm in use.
- `target_group_name` (String) Name of the target group attached to this listener.
- `policy_name` (String) Name of the routing policy (ALB only).
- `action` (String) Policy action (ALB only).
- `compare_type` (String) Comparison type for routing rules (ALB only).
- `type` (String) Rule type (ALB only).
- `value` (String) Value matched by the routing rule (ALB only).

---

## Behavior and Usage Notes

- If no load balancer with the given `lb_name` exists in the specified region, Terraform will return an error.
- This data source is read-only — it will never create, modify, or delete a load balancer.
- Useful for referencing a shared or pre-existing load balancer across multiple Terraform configurations or workspaces.
- For ALB instances, the `listeners` block will include routing policy fields; for NLB instances those fields will be empty strings.
---
page_title: "krutrim_target_group Data Source - terraform-provider-krutrim"
subcategory: "Load Balancer"
description: |-
  Retrieves details of an existing Target Group in Krutrim Cloud.
---

# krutrim_target_group (Data Source)

Provides information about an existing Target Group.

This data source allows you to fetch target group details by name and region, useful when referencing a target group created outside the current Terraform configuration or managed in a separate workspace.

## Example Usage

```hcl
data "krutrim_target_group" "example" {
  name   = "tg-alb"
  region = "In-Bangalore-1"
}
```

## Schema

### Required

- `name` (String) Name of the target group to look up.
- `region` (String) Region where the target group exists (e.g., `"In-Bangalore-1"`).

### Read-Only

- `id` (String) UUID of the target group.
- `vpc_id` (String) ID of the VPC the target group is associated with.
- `members` (List of Object) List of backend members registered in the target group. See [Members Schema](#members-schema) below.
- `health_monitor` (Object) Health check configuration for the target group. See [Health Monitor Schema](#health-monitor-schema) below.

---

### Members Schema

Each object in the `members` list exposes the following read-only fields:

- `name` (String) Display name of the member (typically the VM name).
- `address` (String) Private IP address of the backend instance.
- `protocol_port` (Number) Port on the backend instance receiving traffic.
- `weight` (Number) Relative weight assigned to this member for traffic distribution.

---

### Health Monitor Schema

The `health_monitor` object exposes the following read-only fields:

- `type` (String) Protocol used for health checks (`"HTTP"` or `"TCP"`).
- `delay` (Number) Interval in seconds between health checks.
- `timeout` (Number) Seconds to wait before marking a health check as failed.
- `max_retries` (Number) Consecutive failures before marking a member unhealthy.
- `url_path` (String) HTTP path used for health checks. Populated when `type = "HTTP"`.

---

## Behavior and Usage Notes

- If no target group with the given `name` exists in the specified region, Terraform will return an error.
- This data source is read-only — it will never create, modify, or delete a target group.
- Useful for attaching a pre-existing target group to a new load balancer without re-declaring the group.
