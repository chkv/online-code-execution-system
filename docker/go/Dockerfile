FROM golang:1.25-alpine

# Create non-root user
RUN adduser -D -u 1000 sandbox

WORKDIR /workspace
RUN chmod 777 /workspace

# Pre-warm Go module cache to speed up builds
RUN mkdir -p /var/cache/go-cache && chmod -R 777 /var/cache/go-cache

ENV GOCACHE=/var/cache/go-cache
ENV GOPATH=/tmp/gopath

USER 1000

CMD ["go", "version"]
