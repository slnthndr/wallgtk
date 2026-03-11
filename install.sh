#!/usr/bin/env bash
set -e

APP_NAME="wallgtk"
PREFIX="${PREFIX:-/usr/local}"
BINDIR="$PREFIX/bin"

echo "==> Building $APP_NAME"
go build -o "$APP_NAME"

echo "==> Installing to $BINDIR (sudo may be required)"
sudo install -Dm755 "$APP_NAME" "$BINDIR/$APP_NAME"

echo "==> Done"
echo "Run: $APP_NAME"
