FROM golang:1.25-alpine

RUN adduser -D -u 1000 sandbox

ENV GOCACHE=/var/cache/go-cache \
    GOPATH=/tmp/gopath \
    CGO_ENABLED=0 \
    GOFLAGS="-p=1" \
    GOMAXPROCS=2

WORKDIR /workspace

RUN mkdir -p /var/cache/go-cache /tmp/gopath && \
    chown -R sandbox:sandbox /workspace /var/cache/go-cache /tmp/gopath

USER sandbox

# Populate the build cache while creating the image.
RUN go build std

CMD ["go", "version"]
