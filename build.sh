#!/usr/bin/env bash

mkdir -p build/bin

go build -o build/bin/api ./cmd/api
