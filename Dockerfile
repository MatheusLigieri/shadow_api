FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

# Install goose (version compatible with Go 1.23)
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.22.1

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /build/api .
COPY --from=builder /go/bin/goose .
COPY config.yaml .
COPY openapi.yaml .
COPY migrations ./migrations

EXPOSE 8080

CMD ["./api"]
