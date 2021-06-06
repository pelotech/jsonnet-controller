FROM golang:1.16 as kubecfg-builder

WORKDIR /workspace

# Set the architecture
ARG ARCH=amd64
ENV ARCH=${ARCH}

RUN git clone https://github.com/tinyzimmer/kubecfg && \
        cd kubecfg && git checkout operator-poc

RUN cd kubecfg && GOOS=linux GOARCH=${ARCH} make

# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace

# Set the architecture
ARG ARCH=amd64
ENV ARCH=${ARCH}

# Retrieve the latest kubectl version
RUN    curl -LO "https://dl.k8s.io/release/`curl -L -s https://dl.k8s.io/release/stable.txt`/bin/linux/${ARCH}/kubectl" \
    && curl -LO "https://dl.k8s.io/`curl -L -s https://dl.k8s.io/release/stable.txt`/bin/linux/${ARCH}/kubectl.sha256" \
    && echo "`cat kubectl.sha256` kubectl" | sha256sum --check \
    && chmod +x kubectl

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/kubectl .
COPY --from=kubecfg-builder /workspace/kubecfg/kubecfg .
USER 65532:65532

ENTRYPOINT ["/manager"]
