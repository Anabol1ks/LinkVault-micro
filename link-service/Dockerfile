FROM golang:1.24
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o linkvault-link ./cmd/main.go

CMD ["./linkvault-link"]