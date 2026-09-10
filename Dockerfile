FROM golang:1.25

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o checkout-api ./cmd/api

CMD ["./checkout-api"]
# definition of docker image
# docker build .