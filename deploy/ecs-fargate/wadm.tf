resource "aws_ecs_task_definition" "wadm" {
  family                   = "wadm"
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.wadm_cpu
  memory                   = var.wadm_memory
  skip_destroy             = true

  container_definitions = jsonencode([
    {
      name        = "wadm"
      image       = var.wadm_image
      cpu         = var.wadm_cpu
      memory      = var.wadm_memory
      networkMode = "awsvpc"
      essential   = true
      environment = [
        { name = "WADM_NATS_SERVER", value = "nats.cluster.wasmcloud:4222" },
        { name = "WADM_STRUCTURED_LOGGING_ENABLED", value = "true" }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = "/ecs/wasmcloud"
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "wadm"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "wadm" {
  name            = "wadm"
  cluster         = aws_ecs_cluster.wasmcloud.id
  task_definition = aws_ecs_task_definition.wadm.arn
  desired_count   = var.wadm_count
  launch_type     = "FARGATE"

  network_configuration {
    security_groups = [aws_security_group.wadm_task.id]
    subnets         = aws_subnet.private[*].id
  }

  service_registries {
    registry_arn = aws_service_discovery_service.wadm.arn
  }

  depends_on = [aws_iam_role_policy_attachment.ecs-task-execution-role-policy-attachment]
}

resource "aws_service_discovery_service" "wadm" {
  name = "wadm"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.wasmcloud.id

    dns_records {
      ttl  = 60
      type = "A"
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }
}

resource "aws_security_group" "wadm_task" {
  name        = "wadm-task"
  description = "wadm taskgroup security group"
  vpc_id      = aws_vpc.wasmcloud.id

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }
}

