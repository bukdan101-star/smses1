FROM golang:1.20-alpine

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main ./cmd/server

# Expose port
EXPOSE 3000

# Run the application
CMD ["./main"]