
# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Install git, gcc, and other dependencies required for Go modules and CGO
RUN apk add --no-cache git gcc musl-dev sqlite-dev curl make

# Set the working directory inside the builder container
WORKDIR /src

# Copy the Go source code to the builder container
COPY . .

# Build the Go application
RUN CGO_ENABLED=1 GOOS=linux go build -o SuhaibMessageQueue

# Stage 2: Create the final image
FROM alpine:latest

# Set the working directory in the container
WORKDIR /app

# Create a directory for the database
VOLUME /db

# Copy the binary from the builder stage to the working directory
COPY --from=builder /src/SuhaibMessageQueue .

# Make the binary executable
RUN chmod +x SuhaibMessageQueue

# Specify the command to run when the container starts
CMD ["./SuhaibMessageQueue"]