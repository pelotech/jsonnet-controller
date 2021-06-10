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
    && GOOS=linux GOARCH=${ARCH} GO_LDFLAGS="-s -w" make \
    && upx -9 kubecfg

# Build the manager binary
FROM base-builder as builder

WORKDIR /workspace

# Set the architecture
ARG ARCH=amd64
ENV ARCH=${ARCH}

# Retrieve the latest kubectl version
RUN    curl -LO "https://dl.k8s.io/release/`curl -L -s https://dl.k8s.io/release/stable.txt`/bin/linux/${ARCH}/kubectl" \
    && curl -LO "https://dl.k8s.io/`curl -L -s https://dl.k8s.io/release/stable.txt`/bin/linux/${ARCH}/kubectl.sha256" \
    && echo "`cat kubectl.sha256` kubectl" | sha256sum --check \
    && chmod +x kubectl \
    && upx -9 kubectl

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

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags="-s -w" -a -o manager main.go \
        && upx -9 manager

# Alpine for small base and ca-certificates
FROM alpine:latest

COPY --from=kubecfg-builder /workspace/kubecfg/kubecfg .
COPY --from=builder /workspace/kubectl .
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
