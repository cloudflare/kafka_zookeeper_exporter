FROM golang:1.8.3-alpine

COPY . /go/src/github.com/cloudflare/kafka-zookeeper-exporter

ARG VERSION

RUN go install \
    -ldflags "-X main.version=${VERSION:-dev}" \
    github.com/cloudflare/kafka-zookeeper-exporter && \
    rm -fr /go/src

EXPOSE 8080

CMD ["kafka-zookeeper-exporter"]
