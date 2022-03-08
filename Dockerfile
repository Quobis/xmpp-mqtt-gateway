# syntax=docker/dockerfile:1

FROM golang:1.16-alpine

WORKDIR /gateway

COPY go.mod ./
COPY go.sum ./

RUN go mod download 

COPY *.go ./
COPY config.toml.sample ./
COPY *.txt ./
COPY certs/* ./certs/

RUN go build -o ./xmpp-mqtt-gateway

CMD [ "./xmpp-mqtt-gateway", "config.toml.sample"]
