FROM golang:1.16.4-alpine3.13 AS builder

WORKDIR /bitcandle

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bitcandle

FROM scratch
WORKDIR /data
COPY --from=builder /bitcandle/bitcandle /bitcandle
ENTRYPOINT ["/bitcandle"]