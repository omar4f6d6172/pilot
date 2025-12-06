#!/bin/bash
set -e

# 1. Build your CLI (Running on Arch, targeting Linux)
echo "🔨 Building CLI..."
GOOS=linux go build -o bin/pilot .

# 2. Create the 'Test Runner Service'
# This service file tells systemd what to do when the container boots.
# It runs the tests and then shuts down the machine.
mkdir -p bin
cat <<EOF > bin/test-runner.service
[Unit]
Description=Thesis Test Runner
After=postgresql.service network.target
Requires=postgresql.service

[Service]
Type=idle
WorkingDirectory=/app
# Run bats, allow it to fail (so we can see output), then poweroff
ExecStart=/bin/bash -c "bats tests/integration.bats; echo Exit Code: $?"
StandardOutput=journal+console
ExecStopPost=/usr/bin/poweroff

[Install]
WantedBy=multi-user.target
EOF

echo "🚀 Booting Ephemeral Debian Instance..."

# 3. Run Systemd Nspawn
# -D: Directory of the OS (updated for Debian)
# -x: Ephemeral (changes are lost on exit)
# --bind: Mount code
# --bind: Inject the service file
sudo systemd-nspawn \
    -D ./debian-nspawn \
    --ephemeral \
    --bind=$(pwd):/app \
    --bind=$(pwd)/bin/test-runner.service:/etc/systemd/system/test-runner.service \
    --bind=$(pwd)/bin/test-runner.service:/etc/systemd/system/multi-user.target.wants/test-runner.service \
    --boot
