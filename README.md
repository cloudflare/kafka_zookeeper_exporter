# Kafka ZooKeeper Exporter

A daemon that exposes Kafka cluster state stored in [ZooKeeper](https://kafka.apache.org/documentation/#zk).

## Building

    go get -u github.com/cloudflare/kafka_zookeeper_exporter
    cd $GOPATH/src/github.com/cloudflare/kafka_zookeeper_exporter
    make

## Usage

Start the exporter

    ./kafka-zookeeper-exporter <flags>

To see the list of avaiable flags run

    ./kafka-zookeeper-exporter -h

Send a request to collect metrics

    curl localhost:9381/kafka?zookeeper=10.0.0.1:2181&chroot=/kafka/cluster&topics=mytopic1,mytopic2

Where:

* zookeeper - address of the ZooKeeper used for Kafka, can be multiple addresses separated by comma
* chroot - path inside ZooKeeper where Kafka cluster data is stored
* topics - optional, list of topics to collect metrics for, if empty or missing then all topics will be collected
