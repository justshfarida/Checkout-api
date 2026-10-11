# ---- Build stage: full Go toolchain. Used to compile, never shipped. ----
FROM golang:1.26.9 AS build
WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binary (no C libraries needed at runtime), no local paths or debug
# symbols baked in.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/checkout-api ./cmd/api

# ---- Runtime stage: only the binary. ----
# distroless/static: no shell, no package manager, no source code, no .env.
# Includes CA certificates (needed for HTTPS calls to Xsolla) and a nonroot user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/checkout-api /checkout-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/checkout-api"]
