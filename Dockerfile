FROM golang:latest AS builder

WORKDIR /app

COPY ./ ./
RUN make build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
WORKDIR /app
COPY --from=builder /app/kiali-mcp-server /app/kiali-mcp-server
USER 65532:65532
ENTRYPOINT ["/app/kiali-mcp-server", "--port", "8080"]

EXPOSE 8080
