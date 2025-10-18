FROM golang:1.25-alpine3.22

# Install dependencies
RUN apk add --no-cache git ca-certificates sqlite

WORKDIR /app
    
# Copy go mod files first for better caching
COPY ./server/go.mod ./server/go.sum ./
RUN go mod download

# Copy source code
COPY . .

WORKDIR /app/server

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main . 

# Expose port
EXPOSE 5000

CMD ["./main"]