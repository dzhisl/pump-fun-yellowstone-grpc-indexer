# Use the official Golang image
FROM golang:1.23.4-alpine

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules and Sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the entire project
COPY . .

# Build the Go app
RUN go build -o main cmd/main.go

# Run the Go app
CMD ["./main"]
