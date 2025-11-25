FROM golang:1.25-alpine AS builder

RUN apk add --no-cache curl

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY src/ ./src/

RUN CGO_ENABLED=0 GOOS=linux go build -o /omniauth ./src/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /omniauth .

EXPOSE 8080

# For development, keep the source and Go tools available
CMD ["./omniauth"]
