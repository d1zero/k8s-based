# Stage 1: Build the Go application
FROM golang:1.24.3-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install necessary build dependencies for PostgreSQL
RUN apk add --no-cache git gcc musl-dev

# Copy the Go module files
COPY go.mod ./
COPY go.sum ./

# Download Go modules
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application with static linking for PostgreSQL
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# Stage 2: Create a minimal runtime image
FROM alpine:latest

# Install runtime dependencies for PostgreSQL
RUN apk add --no-cache libpq

# Set the working directory
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose the port the application listens on
EXPOSE 8080

# Define environment variables for PostgreSQL connection
# These can be overridden when running the container
ENV DB_HOST=postgres
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=postgres
ENV DB_NAME=todos

# Command to run the executable
CMD ["./main"]