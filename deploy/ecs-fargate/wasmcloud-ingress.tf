resource "aws_ecs_task_definition" "wasmcloud_ingress" {
  count                    = var.wasmcloud_enable_ingress ? 1 : 0
  family                   = "wasmcloud-ingress"
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
        { name = "WASMCLOUD_LABEL_role", value = "ingress" }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = "/ecs/wasmcloud"
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ingress"
        }
      }
      portMappings = [
        { protocol = "tcp", containerPort = 8080, hostPort = 8080 }
      ]
    }
  ])
}

resource "aws_ecs_service" "wasmcloud_ingress" {
  count           = var.wasmcloud_enable_ingress ? 1 : 0
  name            = "wasmcloud-ingress"
  cluster         = aws_ecs_cluster.wasmcloud.id
  task_definition = one(aws_ecs_task_definition.wasmcloud_ingress[*].arn)
  desired_count   = var.wasmcloud_public_ingress_count
  launch_type     = "FARGATE"

  network_configuration {
    security_groups = aws_security_group.wasmcloud_ingress[*].id
    subnets         = aws_subnet.private[*].id
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.wasmcloud_public.id
    container_name   = "wasmcloud"
    container_port   = 8080
  }


  depends_on = [aws_iam_role_policy_attachment.ecs-task-execution-role-policy-attachment]
}


resource "aws_security_group" "wasmcloud_ingress" {
  count       = var.wasmcloud_enable_ingress ? 1 : 0
  name        = "wasmcloud-ingress"
  description = "wasmcloud public ingress security group"
  vpc_id      = aws_vpc.wasmcloud.id

  ingress {
    protocol = "tcp"
    # NOTE(lxf): All non-privileged ports
    from_port   = 1024
    to_port     = 65535
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "wasmcloud_public" {
  count       = var.wasmcloud_enable_ingress ? 1 : 0
  name        = "wasmcloud-public"
  description = "controls access to Wasmcloud Ingress ALB"
  vpc_id      = aws_vpc.wasmcloud.id

  ingress {
    protocol    = "tcp"
    from_port   = 80
    to_port     = 80
    cidr_blocks = var.wasmcloud_allowed_cidrs
  }

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lb" "wasmcloud_public" {
  count              = var.wasmcloud_enable_ingress ? 1 : 0
  name               = "wasmcloud-public"
  load_balancer_type = "network"
  subnets            = aws_subnet.public[*].id
  security_groups    = aws_security_group.wasmcloud_public[*].id
}

output "wasmcloud_public_lb" {
  value = one(aws_lb.wasmcloud_public[*].dns_name)
}

# NOTE(lxf): Each exposed port needs a target group / listener
resource "aws_lb_target_group" "wasmcloud_public" {
  name        = "wasmcloud"
  port        = 8080
  protocol    = "TCP"
  vpc_id      = aws_vpc.wasmcloud.id
  target_type = "ip"
}

resource "aws_lb_listener" "wasmcloud_public" {
  count             = var.wasmcloud_enable_ingress ? 1 : 0
  load_balancer_arn = one(aws_lb.wasmcloud_public[*].id)
  port              = 80
  protocol          = "TCP"

  default_action {
    target_group_arn = aws_lb_target_group.wasmcloud_public.id
    type             = "forward"
  }
}
