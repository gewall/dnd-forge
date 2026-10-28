# syntax=docker/dockerfile:1

# Fase 2 belum memakai libvips, jadi binary statis (CGO off) di atas distroless.
# Saat upload gambar (govips) masuk di fase 4, ganti ke golang:*-bookworm + libvips-dev
# dan runtime debian:bookworm-slim + libvips42.

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/tools ./cmd/tools

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /out/tools /app/
ENV HTTP_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/app/api", "healthcheck"]
ENTRYPOINT ["/app/api"]
