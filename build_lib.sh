#!/usr/bin/env bash
set -euo pipefail

CORE_DIR="../core"
LIB_DIR="lib"
BUILD_DIR="build_tmp"

echo "[build] Compiling C API wrapper and SDK core..."

mkdir -p "$LIB_DIR" "$BUILD_DIR"

g++ -std=c++17 -O2 -fPIC \
    -I"$CORE_DIR" \
    -c okiff_sdk_capi.cpp \
    -o "$BUILD_DIR/okiff_sdk_capi.o"

g++ -std=c++17 -O2 -fPIC \
    -I"$CORE_DIR" \
    -c "$CORE_DIR/okiff_sdk.cpp" \
    -o "$BUILD_DIR/okiff_sdk.o"

ar rcs "$LIB_DIR/libokiff_sdk.a" \
    "$BUILD_DIR/okiff_sdk_capi.o" \
    "$BUILD_DIR/okiff_sdk.o"

rm -rf "$BUILD_DIR"

echo "[build] Output: $LIB_DIR/libokiff_sdk.a"
echo "[build] Done — you may now remove okiff_sdk_capi.cpp from this directory."