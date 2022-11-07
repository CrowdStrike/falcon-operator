# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace

COPY .git .git
COPY .gitignore .gitignore
COPY hack hack
COPY Makefile Makefile

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY version/ version/
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN make container-build

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /
LABEL name="CrowdStrike Falcon Operator" \
      description="The CrowdStrike Falcon Operator deploys the CrowdStrike Falcon Sensor to protect Kubernetes clusters." \
      maintainer="integrations@crowdstrike.com" \
      summary="The CrowdStrike Falcon Operator" \
      release="0" \
      vendor="CrowdStrike, Inc" \
      version="0.6.1"
COPY LICENSE /licenses/
COPY --from=builder /workspace/manager .

RUN microdnf update -y && microdnf clean all && rm -rf /var/cache/yum/*
USER 65532:65532

ENTRYPOINT ["/manager"]
