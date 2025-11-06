FROM docker.io/golang:1.25.4 AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o feed-renderer


FROM gcr.io/distroless/static-debian12:nonroot

EXPOSE 8000
WORKDIR /app

COPY --from=builder /app/feed-renderer /app/feed-renderer
COPY static /app/static

CMD ["/app/feed-renderer"]
