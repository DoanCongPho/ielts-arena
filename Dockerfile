# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Copy just the dependency manifests first so `go mod download` is cached
# by Docker as its own layer — it only reruns when go.mod/go.sum change,
# not on every source edit.
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source and compile.
COPY . .
# CGO_ENABLED=0 -> static binary, no libc dependency, so it runs unmodified
# on the minimal alpine base below (and would even run on `scratch`).
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# RUN rm -rf /src/*
# ---- runtime stage ----
# Small base: no compiler, no build tools, just enough to run the binary.
FROM alpine:3.20

# The Go backend calls the OpenAI API over HTTPS (internal/platform/llm) —
# without ca-certificates, TLS certificate verification fails.
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/api ./api
# Served at runtime via http.FileServer(http.Dir("internal/assets")) in
# cmd/api/main.go — a relative path read from disk, NOT compiled into the
# binary, so it must be copied alongside it explicitly.
COPY --from=builder /src/internal/assets ./internal/assets

EXPOSE 8080
CMD ["./api"]
