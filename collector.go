package main

import (
	"fmt"
	"strings"
	"sync"
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
	zookeeper string
	chroot    string
	topics    []string
	timeout   time.Duration
	metrics   zkMetrics
}

func newCollector(zookeeper string, chroot string, topics []string) *collector {
	staticLabels := prometheus.Labels{
		"zookeeper": zookeeper,
		"chroot":    chroot,
	}
	return &collector{
		zookeeper: zookeeper,
		chroot:    chroot,
		topics:    topics,
		timeout:   *zkTimeout,
		metrics: zkMetrics{
			topicPartitions: prometheus.NewDesc(
				"kafka_topic_partition_count",
				"Number of partitions on this topic",
				[]string{"topic"},
				staticLabels,
			),
			partitionUsesPreferredReplica: prometheus.NewDesc(
				"kafka_topic_partition_leader_is_preferred",
				"1 if partition is using the preferred broker",
				[]string{"topic", "partition"},
				staticLabels,
			),
			partitionLeader: prometheus.NewDesc(
				"kafka_topic_partition_leader",
				"1 if the node is the leader of this partition",
				[]string{"topic", "partition", "replica"},
				staticLabels,
			),
			partitionReplicaCount: prometheus.NewDesc(
				"kafka_topic_partition_replica_count",
				"Total number of replicas for this topic",
				[]string{"topic", "partition"},
				staticLabels,
			),
			partitionISR: prometheus.NewDesc(
				"kafka_topic_partition_replica_in_sync",
				"1 if replica is in sync",
				[]string{"topic", "partition", "replica"},
				staticLabels,
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

	log.Debugf("Connecting to %s, chroot=%s timeout=%s", c.zookeeper, config.Chroot, config.Timeout)
	client, err := kazoo.NewKazoo(strings.Split(c.zookeeper, ","), &config)
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

	wg := sync.WaitGroup{}
	wg.Add(len(topics))
	for _, topic := range topics {
		go func(cl *collector, cz chan<- prometheus.Metric, t *kazoo.Topic) {
			defer wg.Done()
			if len(cl.topics) > 0 && !stringInSlice(t.Name, cl.topics) {
				// skip topic if it's not on the list of topic to collect
				log.Debugf("Skipping topic '%s', not in list: %s [%d]", t.Name, cl.topics, len(cl.topics))
				return
			}
			c.topicMetrics(ch, t)
		}(c, ch, topic)
	}
	wg.Wait()
}
