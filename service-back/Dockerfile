# Use the official Golang image as a base
FROM golang:latest AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY db.go kafka.go model.go orders.go main.go

# Build the Go app
RUN go build -o order_service .

# Start a new stage from scratch
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/order_service .

# Command to run the executable
CMD ["./order_service"]