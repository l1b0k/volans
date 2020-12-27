FROM golang:1.15 as builder
WORKDIR /go/src/github.com/l1b0k/volans/
COPY . .
RUN go env -w GOPROXY=https://goproxy.io,direct && \
    go build -o volans main.go

FROM centos:8.3.2011
COPY --from=builder /go/src/github.com/l1b0k/volans/main /usr/bin/main
ENTRYPOINT ["/usr/bin/volans"]

