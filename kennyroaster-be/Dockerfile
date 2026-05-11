# ---- Build stage ----
FROM golang:1.25 AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /src

# Copy module files from repo root
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the rest of the source
COPY . .

ENV GOFLAGS=-buildvcs=false

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/app .

# ---- Runtime stage ----
FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/app /app

ENV PORT=8080
ENTRYPOINT ["/app"]