#!/usr/bin/env bash

go mod tidy
golangci-lint run --fix --verbose
