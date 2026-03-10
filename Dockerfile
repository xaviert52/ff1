FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Instalamos GCC para que SQLite compile nativamente
RUN apk add --no-cache gcc musl-dev
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o flows-server ./cmd/server

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/flows-server .
COPY --from=builder /app/flows.db . 
EXPOSE 8080
ENV DB_DRIVER=sqlite
ENV DB_PATH=/app/flows.db
CMD ["./flows-server"]