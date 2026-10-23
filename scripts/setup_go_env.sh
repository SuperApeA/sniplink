#!/bin/bash
set -e

echo "Updating apt sources..."
sudo apt update

echo "Installing Go(1.22)..."
sudo apt install golang-1.22

echo "Setting up Go proxy..."
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

echo "Installing build-essential (make, gcc, g++)..."
sudo apt install -y build-essential

echo "Environment setup complete! Run 'go version' to verify."