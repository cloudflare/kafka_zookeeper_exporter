package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	kazoo "github.com/wvanbergen/kazoo-go"
)

type zkMetrics struct {
	topicPartitions               *prometheus.Desc
	partitionUsesPreferredReplica *prometheus.Desc
	partitionLeader               *prometheus.Desc
	partitionReplicaCount         *prometheus.Desc
	partitionISR                  *prometheus.Desc
}

type collector struct {
	target  string
	chroot  string
	topics  []string
	timeout time.Duration
	metrics zkMetrics
}

func newCollector(target string, chroot string, topics []string) *collector {
	return &collector{
		target:  target,
		chroot:  chroot,
		topics:  topics,
		timeout: *zkTimeout,
		metrics: zkMetrics{
			topicPartitions: prometheus.NewDesc(
				"kafka_topic_partition_count",
				"Number of partitions on this topic",
				[]string{"topic"},
				prometheus.Labels{},
			),
			partitionUsesPreferredReplica: prometheus.NewDesc(
				"kafka_topic_partition_leader_is_preferred",
				"1 if partition is using the preferred broker",
				[]string{"topic", "partition"},
				prometheus.Labels{},
			),
			partitionLeader: prometheus.NewDesc(
				"kafka_topic_partition_leader",
				"1 if the node is the leader of this partition",
				[]string{"topic", "partition", "replica"},
				prometheus.Labels{},
			),
			partitionReplicaCount: prometheus.NewDesc(
				"kafka_topic_partition_replica_count",
				"Total number of replicas for this topic",
				[]string{"topic", "partition"},
				prometheus.Labels{},
			),
			partitionISR: prometheus.NewDesc(
				"kafka_topic_partition_replica_in_sync",
				"1 if replica is in sync",
				[]string{"topic", "partition", "replica"},
				prometheus.Labels{},
			),
		},
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.topicPartitions
	ch <- c.metrics.partitionUsesPreferredReplica
	ch <- c.metrics.partitionLeader
	ch <- c.metrics.partitionReplicaCount
	ch <- c.metrics.partitionISR
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	config := kazoo.Config{
		Chroot:  c.chroot,
		Timeout: c.timeout,
	}

	log.Debugf("Connecting to %s, chroot=%s timeout=%s", c.target, config.Chroot, config.Timeout)
	client, err := kazoo.NewKazoo(strings.Split(c.target, ","), &config)
	if err != nil {
		msg := fmt.Sprintf("Connection error: %s", err)
		log.Error(msg)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("zookeeper_error", msg, nil, nil), err)
		return
	}

	defer client.Close()

	topics, err := client.Topics()
	if err != nil {
		msg := fmt.Sprintf("Error collecting list of topics: %s", err)
		log.Error(msg)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("zookeeper_topic_list_error", msg, nil, nil), err)
		return
	}

	// kafka_zookeeper_up{} 1
	ch <- prometheus.MustNewConstMetric(c.metrics.kafkaUp, prometheus.GaugeValue, 1)

	for _, topic := range topics {
		if len(c.topics) > 0 && !stringInSlice(topic.Name, c.topics) {
			// skip topic if it's not on the list of topic to collect
			log.Debugf("Skipping topic '%s', not in list: %s [%d]", topic.Name, c.topics, len(c.topics))
			continue
		}
		c.topicMetrics(ch, topic)
	}
}
