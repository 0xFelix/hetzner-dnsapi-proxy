# Build binary

FROM golang:alpine AS builder

RUN apk add --update make

WORKDIR /workspace

COPY Makefile .
COPY go.mod .
COPY go.sum .
COPY main.go .
COPY pkg/ pkg/
COPY vendor/ vendor/

ARG VERSION=""
RUN make build VERSION=${VERSION}

# Build image

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /workspace/bin/hetzner-dnsapi-proxy /

EXPOSE 8081
ENTRYPOINT ["/hetzner-dnsapi-proxy"]
