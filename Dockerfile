#Build stage
FROM golang:1.25-alpine AS builder

#install build dependencies
RUN apk add --no-cache git build-base postgresql-dev ca-certificates

#set the working directory
WORKDIR /app

#copy the dependencies first
COPY go.mod go.sum ./

#download dependencies with proper retry
RUN for i in $(seq 1 3);do \
    go mod download && break || sleep 5; \
    done

#copy the source code
COPY . .

#Build the binary code and write the output on some directory
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s"  -o /app/main ./cmd/server

#Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .

CMD [ "./main" ]
