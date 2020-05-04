FROM golang:1.13

RUN apt-get update && apt-get install -y fuse

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main .

CMD ["./main"]
