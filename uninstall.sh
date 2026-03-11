#!/usr/bin/env bash
set -e

APP_NAME="wallgtk"
PREFIX="${PREFIX:-/usr/local}"
BINDIR="$PREFIX/bin"

echo "==> Removing $BINDIR/$APP_NAME"
sudo rm -f "$BINDIR/$APP_NAME"

echo "==> Uninstalled"
