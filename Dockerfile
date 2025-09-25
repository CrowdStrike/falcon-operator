# Build the manager binary
FROM registry.access.redhat.com/ubi8/go-toolset:1.23 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG GOPROXY
ENV GOPROXY=${GOPROXY}

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY version/ version/
COPY api/ api/
COPY internal/controller/ internal/controller/
COPY internal/errors/ internal/errors/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
#
# FIPS documentation: https://developers.redhat.com/articles/2025/01/23/fips-mode-red-hat-go-toolset#validating_fips_mode_capabilities
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -tags \
    "exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp" \
    --ldflags="-X 'github.com/crowdstrike/falcon-operator/version.Version=${VERSION}'" \
    -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8-minimal:8.10
ARG VERSION
WORKDIR /
COPY LICENSE licenses/
COPY --from=builder /etc/pki /etc/pki
COPY --from=builder /workspace/manager .
USER 65532:65532

LABEL name="falcon-operator" \
      vendor="CrowdStrike, Inc" \
      version="${VERSION}" \
      release="1" \
      summary="Crowdstrike Falcon Operator Controller" \
      description="The CrowdStrike Falcon Operator installs CrowdStrike Falcon custom resources on a Kubernetes cluster." \
      maintainer="support@crowdstrike.com"

ENTRYPOINT ["/manager"]
