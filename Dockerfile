FROM golang:1.18-alpine
RUN apk add git

RUN mkdir /data
RUN mkdir /certs

RUN mkdir /app
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy && go mod download && go mod verify

COPY . .

RUN go build -v -o turbo-ledger.
EXPOSE 12000
CMD ["/app/turbo-ledger"]