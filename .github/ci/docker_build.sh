#!/bin/sh

go mod tidy
go build -o abi-app-store
chmod +x abi-app-store
