#!/usr/bin/env bash
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root: sudo $0" >&2
  exit 1
fi

apt-get update -qq
apt-get install -y -qq lvm2 e2fsprogs util-linux
