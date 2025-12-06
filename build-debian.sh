#!/bin/bash
set -e

# Directory where the OS will live
TARGET_DIR="./debian-nspawn"

if [ -d "$TARGET_DIR" ]; then
    echo "✅ Directory exists. Remove it to rebuild."
    exit 0
fi

echo "📦 Downloading minimal Debian (Bookworm)..."

# 1. Run debootstrap
# --variant=minbase: strict minimum
# --include: We add passwd, systemd, dbus, and crucial tools explicitly
sudo debootstrap \
    --variant=minbase \
    --include=systemd,dbus,udev,iproute2,passwd,sudo,curl,procps \
    bookworm \
    $TARGET_DIR \
    http://deb.debian.org/debian/

echo "⚙️  Configuring..."

# 2. Fix the Password (The Fix: Use full path /usr/sbin/chpasswd)
echo "root:root" | sudo chroot $TARGET_DIR /usr/sbin/chpasswd

# 3. Quick Network Fix for systemd-nspawn
# This ensures the container creates a network interface
echo "[Match]
Name=host0

[Network]
DHCP=yes" | sudo tee $TARGET_DIR/etc/systemd/network/80-container-host0.network > /dev/null

# 4. Enable systemd-networkd so internet works inside
sudo chroot $TARGET_DIR systemctl enable systemd-networkd

echo "✅ Done! Minimal Debian is ready in $TARGET_DIR"
