FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o mcp-openapi main.go

FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /app/mcp-openapi .

CMD ["./mcp-openapi"]
