FROM golang:1.18

ENV GOPATH=""
ENV GO111MODULE=on

ADD go.mod go.mod
ADD go.sum go.sum

RUN go mod download

ADD main.go main.go

RUN mkdir -p build
RUN go build -o build

ENTRYPOINT [ "build/docker-volume-backup" ]
