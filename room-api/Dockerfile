FROM golang:1.25 AS build

WORKDIR /src
COPY go.mod go.sum* ./
COPY vendor ./vendor
COPY . .
RUN CGO_ENABLED=1 GOFLAGS=-mod=vendor go build -trimpath -ldflags='-s -w' -o /out/room-api ./cmd/room-api

FROM debian:bookworm-slim

RUN useradd --system --uid 10001 --create-home roomapi
WORKDIR /app
COPY --from=build /out/room-api /app/room-api
RUN mkdir -p /var/lib/room-api && chown -R roomapi:roomapi /var/lib/room-api /app
USER roomapi
EXPOSE 8080
ENTRYPOINT ["/app/room-api"]
