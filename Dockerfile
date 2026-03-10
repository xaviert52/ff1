# STAGE 1: Builder
FROM golang:alpine AS builder
WORKDIR /app
# Instalamos dependencias de compilación
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Compilamos el binario para Linux
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o flows-server ./cmd/server

# STAGE 2: Runtime
FROM alpine:latest
WORKDIR /app
# Certificados para llamadas HTTPS externas (WhatsApp API)
RUN apk add --no-cache ca-certificates
# SÓLO copiamos el binario, NO la base de datos sqlite
COPY --from=builder /app/flows-server .

EXPOSE 8080
CMD ["./flows-server"]