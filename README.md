# Kafka ZooKeeper Exporter

A daemon that exposes Kafka cluster state stored in [ZooKeeper](https://kafka.apache.org/documentation/#zk).

## Motivation

Metrics exported by `kafka_zookeeper_exporter` provide cluster level overview
of the entire cluster and can be used along
[jmx_exporter](https://github.com/prometheus/jmx_exporter) which provides broker
level data. `jmx_exporter` exports what each brokers believes to be true, but
this information can be incorrect in case of a network partition or other split
brain issues. ZooKeeper on the other hand is the source of truth for the entire
cluster configuration and runtime status, so the metrics exported from it are
the best representation of the entire cluster status.

## Metrics

### kafka_topic_partition_count

Number of partitions configured for given topic.

### kafka_topic_partition_replica_count

Number of replicas configured for given partition.

### kafka_topic_partition_leader

This metric will have value `1` for the replica that is currently the leader for
given partition.

### kafka_topic_partition_leader_is_preferred

Each Kafka partition have a list of replicas, the first replica is the preferred
(default) leader. This metric will have value `1` if the current partition
leader is the preferred one.

### kafka_topic_partition_replica_in_sync

This metric will indicate whenever given replica is in sync with the partition
leader.

### kafka_broker_is_controller

This metric will have value `1` for the broker that is currently the cluster
controller.

### kafka_consumers_offsets

The last offset consumed for a given (consumer, topic, partition).

This will only show metrics for legacy consumers that still store their offsets
in Zookeeper.

### kafka_zookeeper_scrape_error

Will have value `1` if there was an error retrieving or processing any of the
data for the current scrape. `0` otherwise.

## Building

    go get -u github.com/cloudflare/kafka_zookeeper_exporter
    cd $GOPATH/src/github.com/cloudflare/kafka_zookeeper_exporter
    make

## Usage

Start the exporter

    ./kafka_zookeeper_exporter <flags>

To see the list of avaiable flags run

    ./kafka_zookeeper_exporter -h

Send a request to collect metrics

    curl localhost:9381/kafka?zookeeper=10.0.0.1:2181&chroot=/kafka/cluster&topic=mytopic1,mytopic2&consumer=myconsumer1,myconsumer2

Where:

* zookeeper - required, address of the ZooKeeper used for Kafka, can be multiple addresses separated by comma
* chroot - path inside ZooKeeper where Kafka cluster data is stored. Has to be omitted if Kafka resides in the root of ZooKeeper.
* topic - optional, list of topics to collect metrics for.
  If empty or missing then all topics will be collected.
* consumer - optional, list of consumers to collect metrics for.
  If empty or missing then all consumers will be collected.

If both topic and consumer are non-empty, then metrics where both are relevant
will only be collected if they match both.

## Prometheus configuration

Example Prometheus scrape job configuration:

    - job_name: kafka_zookeeper_exporter_mycluster
      static_configs:
        - targets:
          # hostname and port `kafka-zookeeper-exporter` is listening on
          - myserver:9381
      metrics_path: /kafka
      scheme: http
      params:
        zookeeper: ['zk1.example.com:2181,zk2.example.com:2181']
        chroot: ['/kafka/mycluster']

This example uses `static_configs` to configure scrape target.
See [Prometheus docs](https://prometheus.io/docs/operating/configuration/) for other
ways to configure it.
