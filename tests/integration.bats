#!/usr/bin/env bats

@test "Check if postgres is running" {
  run systemctl is-active postgresql
  [ "$status" -eq 0 ]
}

@test "Check if pilot binary exists" {
  [ -f /app/bin/pilot ]
}

@test "Check pilot help command" {
  run /app/bin/pilot --help
  [ "$status" -eq 0 ]
}
