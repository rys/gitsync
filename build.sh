#!/usr/local/bin/zsh

GIT_SHA=$(git rev-parse --short HEAD)
GIT_TAG=$(git describe --tags)
GIT_DATE=$(git log -1 --format=%cd --date=short)
BUILD_DATE=$(date -u '+%Y-%m-%d@%H%M')
BUILD_USER=${1:l}

OS=$(uname)
ARCH=$(uname -m)

[[ -z ${GIT_TAG} ]] && GIT_TAG=dev
[[ -z ${BUILD_USER} ]] && BUILD_USER=${USER}

LDFLAGS="-X main.BuildVersion=${GIT_TAG} -X main.BuildDate=${BUILD_DATE} -X main.GitDate=${GIT_DATE} -X main.GitRevision=${GIT_SHA} -X main.BuildUser=${BUILD_USER}"

go get -u
go build -ldflags ${LDFLAGS}