# ---- Build stage: full Go toolchain. Used to compile, never shipped. ----
FROM golang:1.26.9 AS build
WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binaries (no C libraries needed at runtime), no local paths or debug
# symbols baked in. One image, three programs: the API, the email worker and the analytics worker.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/checkout-api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/notification-worker ./cmd/notification_worker \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/analytics-worker ./cmd/analytics_worker

# ---- Runtime stage: only the binary. ----
# distroless/static: no shell, no package manager, no source code, no .env.
# Includes CA certificates (needed for HTTPS calls to Xsolla) and a nonroot user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/checkout-api /checkout-api
COPY --from=build /out/notification-worker /notification-worker
COPY --from=build /out/analytics-worker /analytics-worker
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/checkout-api"]
