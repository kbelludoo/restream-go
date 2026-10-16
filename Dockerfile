# Etapa de compilação
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o restream .

# Imagem final mínima (~15 MB)
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/restream .
COPY config.example.json .

EXPOSE 1935 3008

ENTRYPOINT ["./restream"]
