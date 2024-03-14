# Use the official Go image as the base image
FROM golang:1.22-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download and install the Go dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the port that the application listens on
EXPOSE 8080

# Set the entry point for the container
CMD ["./main"]
