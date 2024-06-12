

# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
RUN apt-get update && \
    wget https://github.com/mongodb-js/mongosh/releases/download/v1.6.2/mongodb-mongosh_1.6.2_amd64.deb && \
    apt-get install ./mongodb-mongosh_1.6.2_amd64.deb && \
    apt-get install -y libkrb5-3 \
        libssl1.1 \
        libcurl4 \
        libgssapi-krb5-2 \
        libuuid1 \
        libc6 \
        libgcc1 \
        libcom-err2 \
        libcomerr2 \
        keyutils && \
    rm -rf /var/lib/apt/lists/*


COPY go.mod go.sum ./
RUN \
	echo ">> Downloading go modules..." && \
	go mod download

COPY . .

RUN go build -ldflags="-s -w" -o middleware-operator cmd/middleware/main.go



# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/base:nonroot
FROM gcr.io/distroless/base:debug
WORKDIR /
COPY --from=builder /workspace/middleware-operator .
COPY --from=builder /usr/bin/mongosh /usr/bin
COPY --from=builder /usr/lib/x86_64-linux-gnu/* /usr/lib/x86_64-linux-gnu
COPY --from=builder /lib/x86_64-linux-gnu/libcom_err.so.2 /usr/lib/x86_64-linux-gnu
COPY --from=builder /lib/x86_64-linux-gnu/libgcc_s.so.1 /usr/lib/x86_64-linux-gnu
COPY --from=builder /lib/x86_64-linux-gnu/libkeyutils.so.1 /usr/lib/x86_64-linux-gnu


EXPOSE 8080

ENTRYPOINT ["/middleware-operator"]