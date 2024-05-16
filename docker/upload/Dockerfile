

# Build the manager binary
FROM golang:1.22 as builder

WORKDIR /workspace
COPY go.mod go.sum ./
RUN \
  echo ">> Downloading go modules..." && \
  go mod download

COPY / ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o upload cmd/upload/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/base:nonroot
FROM alpine:latest
WORKDIR /
COPY --from=builder /workspace/upload .

ENTRYPOINT ["/upload"]