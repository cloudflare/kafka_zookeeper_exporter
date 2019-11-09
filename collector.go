package main

import (
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
	controller                    *prometheus.Desc
	consumersOffsets              *prometheus.Desc
	zookeeperScrapeError          *prometheus.Desc
}

type collector struct {
	zookeeper string
	chroot    string
	topics    []string
	consumers []string
	timeout   time.Duration
	zkErr     int32
	metrics   zkMetrics
}

func newCollector(zookeeper string, chroot string, topics []string, consumers []string) *collector {
	return &collector{
		zookeeper: zookeeper,
		chroot:    chroot,
		topics:    topics,
		consumers: consumers,
		timeout:   *zkTimeout,
		zkErr:     0,
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
				"Total number of replicas for this partition",
				[]string{"topic", "partition"},
				prometheus.Labels{},
			),
			partitionISR: prometheus.NewDesc(
				"kafka_topic_partition_replica_in_sync",
				"1 if replica is in sync",
				[]string{"topic", "partition", "replica"},
				prometheus.Labels{},
			),
			controller: prometheus.NewDesc(
				"kafka_broker_is_controller",
				"1 if the broker is the controller of this cluster",
				[]string{"broker"},
				prometheus.Labels{},
			),
			consumersOffsets: prometheus.NewDesc(
				"kafka_consumers_offsets",
				"Last offset consumed",
				[]string{"consumer", "topic", "partition"},
				prometheus.Labels{},
			),
			zookeeperScrapeError: prometheus.NewDesc(
				"kafka_zookeeper_scrape_error",
				"1 if there were any errors retrieving data for this scrape, 0 otherwise",
				[]string{},
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
	ch <- c.metrics.controller
	ch <- c.metrics.consumersOffsets
	ch <- c.metrics.zookeeperScrapeError
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	config := kazoo.Config{
		Chroot:  c.chroot,
		Timeout: c.timeout,
	}

	log.Debugf("Connecting to %s, chroot=%s timeout=%s", c.zookeeper, config.Chroot, config.Timeout)
	client, err := kazoo.NewKazoo(strings.Split(c.zookeeper, ","), &config)
	if err != nil {
		log.Errorf("Connection error: %s", err)
		// If we can't connect, there's nothing left to do but return the error to the client
		ch <- prometheus.MustNewConstMetric(c.metrics.zookeeperScrapeError, prometheus.GaugeValue, 1)
		return
	}
	defer client.Close()

	c.clusterMetrics(ch, client)

	wg := sync.WaitGroup{}
	topics, err := client.Topics()
	if err != nil {
		log.Errorf("Error collecting list of topics: %s", err)
		c.zkErr = 1
	} else {
		for _, topic := range topics {
			if len(c.topics) > 0 && !stringInSlice(topic.Name, c.topics) {
				// skip topic if it's not on the list of topic to collect
				log.Debugf("Skipping topic '%s', not in list: %s [%d]", topic.Name, c.topics, len(c.topics))
			} else {
				wg.Add(1)
				go func(t *kazoo.Topic) {
					defer wg.Done()
					c.topicMetrics(ch, t)
				}(topic)
			}
		}
	}

	consumers, err := client.Consumergroups()
	if err != nil {
		log.Errorf("Error collecting list of consumers: %s", err)
		c.zkErr = 1
	} else {
		for _, consumer := range consumers {
			if len(c.consumers) > 0 && !stringInSlice(consumer.Name, c.consumers) {
				// skip consumer if it's not on the list of consumers to collect
				log.Debugf("Skipping consumer '%s', not in list: %s [%d]", consumer.Name, c.consumers, len(c.consumers))
			} else {
				wg.Add(1)
				go func(cg *kazoo.Consumergroup) {
					defer wg.Done()
					c.consumerMetrics(ch, cg)
				}(consumer)
			}
		}
	}

	wg.Wait()

	ch <- prometheus.MustNewConstMetric(c.metrics.zookeeperScrapeError, prometheus.GaugeValue, float64(c.zkErr))
}
