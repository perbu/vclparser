#!/bin/bash
set -e
set -x
# Test script to validate all VCL files in testdata directory with varnishd
# Usage: ./test_vcl_files.sh

VARNISHD="/opt/homebrew/sbin/varnishd"

echo "Testing VCL files with varnishd..."
echo "=================================="

CUR=$(pwd)
mkdir -p "${CUR}/vcltmp"
# Test each .vcl file in the current directory
for vcl_file in *.vcl; do
    if [[ -f "$vcl_file" ]]; then
        echo -n "Testing $vcl_file"
        # Run varnishd with the VCL file
        "$VARNISHD" -j none -n "$CUR/vcltmp" -C -f "$CUR/$vcl_file"
    fi
done

rm -rf $cur/vcltmp
echo OK