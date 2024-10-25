resource "aws_ecs_task_definition" "wasmcloud_workload" {
  family                   = "wasmcloud-workload"
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.wasmcloud_cpu
  memory                   = var.wasmcloud_memory
  skip_destroy             = true

  container_definitions = jsonencode([
    {
      name        = "wasmcloud"
      image       = var.wasmcloud_image
      cpu         = var.wasmcloud_cpu
      memory      = var.wasmcloud_memory
      networkMode = "awsvpc"
      essential   = true
      environment = [
        { name = "WASMCLOUD_NATS_HOST", value = "nats.cluster.wasmcloud" },
        { name = "WASMCLOUD_STRUCTURED_LOGGING_ENABLED", value = "true" },
        { name = "WASMCLOUD_LABEL_role", value = "workload" }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = "/ecs/wasmcloud"
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "workload"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "wasmcloud_workload" {
  name            = "wasmcloud-workload"
  cluster         = aws_ecs_cluster.wasmcloud.id
  task_definition = aws_ecs_task_definition.wasmcloud_workload.arn
  desired_count   = var.wasmcloud_workload_min_count
  launch_type     = "FARGATE"

  network_configuration {
    security_groups = [aws_security_group.wasmcloud_workload.id]
    subnets         = aws_subnet.private[*].id
  }

  depends_on = [aws_iam_role_policy_attachment.ecs-task-execution-role-policy-attachment]

  lifecycle {
    ignore_changes = [desired_count]
  }
}


resource "aws_security_group" "wasmcloud_workload" {
  name        = "wasmcloud-workload"
  description = "wasmcloud workload security group"
  vpc_id      = aws_vpc.wasmcloud.id

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_appautoscaling_target" "wasmcloud_workload" {
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_cluster.wasmcloud.name}/${aws_ecs_service.wasmcloud_workload.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  min_capacity       = var.wasmcloud_workload_min_count
  max_capacity       = var.wasmcloud_workload_max_count
}

resource "aws_appautoscaling_policy" "wasmcloud_workload_up" {
  name               = "wasmcloud-workload-up"
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_cluster.wasmcloud.name}/${aws_ecs_service.wasmcloud_workload.name}"
  scalable_dimension = "ecs:service:DesiredCount"

  step_scaling_policy_configuration {
    adjustment_type         = "ChangeInCapacity"
    cooldown                = 60
    metric_aggregation_type = "Maximum"

    step_adjustment {
      metric_interval_lower_bound = 0
      scaling_adjustment          = 1
    }
  }

  depends_on = [aws_appautoscaling_target.wasmcloud_workload]
}

resource "aws_cloudwatch_metric_alarm" "wasmcloud_workload_cpu_high" {
  alarm_name          = "wasmcloud-workload-cpu-high"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = "60"
  statistic           = "Average"
  threshold           = "85"

  dimensions = {
    ClusterName = aws_ecs_cluster.wasmcloud.name
    ServiceName = aws_ecs_service.wasmcloud_workload.name
  }

  alarm_actions = [aws_appautoscaling_policy.wasmcloud_workload_up.arn]
}

resource "aws_appautoscaling_policy" "wasmcloud_workload_down" {
  name               = "wasmcloud-workload-down"
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_cluster.wasmcloud.name}/${aws_ecs_service.wasmcloud_workload.name}"
  scalable_dimension = "ecs:service:DesiredCount"

  step_scaling_policy_configuration {
    adjustment_type         = "ChangeInCapacity"
    cooldown                = 60
    metric_aggregation_type = "Maximum"

    step_adjustment {
      metric_interval_lower_bound = 0
      scaling_adjustment          = -1
    }
  }

  depends_on = [aws_appautoscaling_target.wasmcloud_workload]
}

resource "aws_cloudwatch_metric_alarm" "wasmcloud_workload_cpu_low" {
  alarm_name          = "wasmcloud-workload-cpu-low"
  comparison_operator = "LessThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = "60"
  statistic           = "Average"
  threshold           = "10"

  dimensions = {
    ClusterName = aws_ecs_cluster.wasmcloud.name
    ServiceName = aws_ecs_service.wasmcloud_workload.name
  }

  alarm_actions = [aws_appautoscaling_policy.wasmcloud_workload_down.arn]
}
