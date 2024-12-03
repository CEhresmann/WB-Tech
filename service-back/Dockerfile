# Use the official Golang image as a base
FROM golang:latest AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./
COPY config/db.json ./
COPY DB/initial.sql ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY db.go kafka.go model.go orders.go main.go ./

# Build the Go app
RUN go build -o order_service .
RUN chmod +x order_service
RUN ls -l /app


# Start a new stage from scratch
FROM scratch

WORKDIR /root/

COPY --from=builder /app/order_service .
COPY --from=builder /app/db.json .
COPY --from=builder /app/initial.sql ./DB/


CMD ["./order_service"]