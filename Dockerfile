FROM golang:1.25

WORKDIR /app

COPY ./ ./

RUN go mod download


RUN go build -o /checkout-api

CMD ["/checkout-api"]
# definition of docker image
# docker build .