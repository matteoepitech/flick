#!/usr/bin/env bash

mkdir -p build/bin

go build -o build/bin/flick-api ./cmd/api
go build -o build/bin/flick-cli ./cmd/cli
