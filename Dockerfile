
# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Install git, gcc, and other dependencies required for Go modules and CGO
RUN apk add --no-cache git gcc musl-dev sqlite-dev curl make

# Set the working directory inside the builder container
WORKDIR /src

# Copy the Go source code to the builder container
COPY . .

# Set environment variables for cross-compilation
ENV CGO_ENABLED=1
ENV GOOS=linux

# Build the Go application
RUN go build -o smq ./cmd/smq

# Stage 2: Create the final image
FROM alpine:latest

# Set the working directory in the container
WORKDIR /app

# Create a directory for the database
VOLUME /db

# Copy the binary from the builder stage to the working directory
COPY --from=builder /src/smq .

# Make the binary executable
RUN chmod +x smq

# Specify the command to run when the container starts
CMD ["./smq"]