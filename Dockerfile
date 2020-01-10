FROM scratch
COPY kafka_zookeeper_exporter /kafka_zookeeper_exporter
CMD ["/kafka_zookeeper_exporter"]