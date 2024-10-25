resource "aws_cloudwatch_log_group" "cb_log_group" {
  name              = "/ecs/wasmcloud"
  retention_in_days = 30

  tags = {
    Name = "wasmcloud"
  }
}

resource "aws_cloudwatch_log_stream" "cb_log_stream" {
  name           = "wasmcloud-log-stream"
  log_group_name = aws_cloudwatch_log_group.cb_log_group.name
}

