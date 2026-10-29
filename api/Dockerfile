# syntax=docker/dockerfile:1

# Binary statis (CGO off) di atas distroless. Tag `nodynamic` wajib: tanpa itu
# gen2brain/webp memakai purego untuk mencari libwebp sistem, binary jadi ter-link
# dinamis, dan container gagal start ("exec /app/api: no such file or directory").

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -tags nodynamic -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -tags nodynamic -trimpath -ldflags="-s -w" -o /out/tools ./cmd/tools

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /out/tools /app/
ENV HTTP_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/app/api", "healthcheck"]
ENTRYPOINT ["/app/api"]
