# syntax=docker/dockerfile:1

ARG GO_VERSION="1.23.7"
ARG BUILDPLATFORM="linux/amd64"
ARG BASE_IMAGE="golang:${GO_VERSION}-alpine"
ARG WASMVM_VERSION="v2.2.1"

FROM --platform=${BUILDPLATFORM} ${BASE_IMAGE}

ARG GIT_COMMIT
ARG GIT_VERSION

# NOTE: add libusb-dev to run with LEDGER_ENABLED=true
RUN set -eux &&\
    apk update &&\
    apk add --no-cache \
    ca-certificates \
    linux-headers \
    build-base \
    cmake \
    git

# install mimalloc for musl
WORKDIR ${GOPATH}/src/mimalloc
RUN set -eux &&\
    git clone --depth 1 --branch v3.0.1 \
        https://github.com/microsoft/mimalloc . &&\
    mkdir -p build &&\
    cd build &&\
    cmake .. &&\
    make -j$(nproc) &&\
    make install

# download dependencies to cache as layer
WORKDIR ${GOPATH}/src/app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go mod download -x

# Cosmwasm - Download correct libwasmvm version and verify checksum
RUN set -eux &&\
    WASMVM_DOWNLOADS="https://github.com/CosmWasm/wasmvm/releases/download/v2.1.4"; \
    wget ${WASMVM_DOWNLOADS}/checksums.txt -O /tmp/checksums.txt; \
    WASMVM_URL="${WASMVM_DOWNLOADS}/libwasmvm_muslc.x86_64.a"; \
    wget ${WASMVM_URL} -O /lib/libwasmvm_muslc.x86_64.a; \
    CHECKSUM=`sha256sum /lib/libwasmvm_muslc.x86_64.a | cut -d" " -f1`; \
    grep ${CHECKSUM} /tmp/checksums.txt; \
    rm /tmp/checksums.txt

# Copy the remaining files
COPY . .

# Build app binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go install \
        -mod=readonly \
        -tags "netgo,muslc" \
        -ldflags " \
            -w -s -linkmode=external -extldflags \
            '-L/go/src/mimalloc/build -lmimalloc -Wl,-z,muldefs -static' \
            -X github.com/cosmos/cosmos-sdk/version.Name='kopi' \
            -X github.com/cosmos/cosmos-sdk/version.AppName='kopid' \
            -X github.com/cosmos/cosmos-sdk/version.Version=${GIT_VERSION} \
            -X github.com/cosmos/cosmos-sdk/version.Commit=${GIT_COMMIT} \
            -X github.com/cosmos/cosmos-sdk/version.BuildTags='netgo,muslc' \
        " \
        -trimpath \
        ./...
