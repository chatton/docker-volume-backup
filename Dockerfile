FROM golang:1.18 as builder

ENV GOPATH=""
ENV GO111MODULE=on

ADD go.mod go.mod
ADD go.sum go.sum

RUN go mod download

ADD main.go main.go

RUN mkdir -p build
RUN go build -o build

FROM busybox:latest

COPY --from=builder /go/build/docker-volume-backup build/docker-volume-backup

ENTRYPOINT [ "docker-volume-backup" ]
