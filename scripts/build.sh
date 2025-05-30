#!/bin/bash

echo "Building CfM ..."

echo "building Server ..."
go build -o bin/cfm-server ./cmd/server

echo "Building client ..."
go build -o bin/cfm ./cmd/cfm

