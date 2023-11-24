# Build the main binary
FROM --platform=$TARGETPLATFORM golang:1.21 as builder
ARG TARGETARCH
ARG TARGETOS

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# Copy the go source
COPY main.go main.go
COPY lib/ lib/

# Build
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -a -o main main.go

# Use distroless as minimal base image to package the main binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/cc:nonroot
WORKDIR /
COPY --from=builder /workspace/main .
COPY --from=alpine/git /usr/bin/git /usr/bin/git
USER 65532:65532

ENTRYPOINT ["/main"]
