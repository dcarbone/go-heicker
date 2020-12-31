#!/usr/bin/env sh

set +ex

BUILD_NAME="heicker"
BUILD_DATE="$(date -u +%Y%m%d@%H%M%S%z)"
BUILD_BRANCH="$(git branch --no-color|awk '/^\*/ {print $2}')"

export GO111MODULE=on
export GOFLAGS="-mod=vendor"

go vet ./...

go build -o="${BUILD_NAME}" -ldflags "\
-X main.BuildName=${BUILD_NAME} \
-X main.BuildDate=${BUILD_DATE} \
-X main.BuildBranch=${BUILD_BRANCH}"