FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY pkg ./pkg
COPY db ./db
COPY scripts ./scripts
COPY integration ./integration

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/dbctl ./cmd/dbctl
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/platform_runtime ./cmd/platform_runtime

FROM alpine:3.21

RUN adduser -D -H -u 10001 appuser
WORKDIR /app

COPY --from=build /out/dbctl /usr/local/bin/dbctl
COPY --from=build /out/platform_runtime /usr/local/bin/platform_runtime
COPY db ./db
COPY scripts ./scripts

USER appuser

# Default command runs long-lived runtime service.
# For migrations in Sliplane, override command to:
#   dbctl migrate up --dir /app/db/migrations
CMD ["platform_runtime", "serve"]
