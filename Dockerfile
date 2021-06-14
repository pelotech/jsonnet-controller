FROM golang:1.16 as base-builder

RUN apt-get update && apt-get install -y upx

FROM base-builder as kubecfg-builder

WORKDIR /workspace

# Set the architecture
ARG ARCH=amd64
ENV ARCH=${ARCH}

RUN git clone https://github.com/tinyzimmer/kubecfg && \
        cd kubecfg && git checkout operator-poc

RUN cd kubecfg \
    && GOOS=linux GOARCH=${ARCH} GO_LDFLAGS="-s -w" make

# Build the manager binary
FROM base-builder as builder

WORKDIR /workspace

# Set the architecture
ARG ARCH=amd64
ENV ARCH=${ARCH}

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags="-s -w" -a -o manager main.go \
        && upx -9 manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=kubecfg-builder /workspace/kubecfg/kubecfg .
USER 65532:65532

ENTRYPOINT ["/manager"]