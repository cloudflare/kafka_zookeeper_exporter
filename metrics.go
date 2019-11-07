package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	kazoo "github.com/wvanbergen/kazoo-go"
)

func (c *collector) clusterMetrics(ch chan<- prometheus.Metric, client *kazoo.Kazoo) {
	controller, err := client.Controller()
	if err != nil {
		msg := fmt.Sprintf("Error collecting cluster controller broker ID: %s", err)
		log.Error(msg)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("zookeeper_controller_id_error", msg, nil, nil), err)
		return
	}

	// kafka_broker_is_controller{broker="123"} 1
	ch <- prometheus.MustNewConstMetric(
		c.metrics.controller,
		prometheus.GaugeValue, 1,
		fmt.Sprint(controller),
	)
}

func (c *collector) topicMetrics(ch chan<- prometheus.Metric, topic *kazoo.Topic) {
	// per partition metrics
	partitions, err := topic.Partitions()
	if err != nil {
		msg := fmt.Sprintf("Error collecting list of partitions on '%s' topics: %s", topic.Name, err)
		log.Errorf(msg)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("zookeeper_topic_partitions_error", msg, nil, nil), err)
		return
	}
	// kafka_topic_partition_count{topic="name"} 13
	ch <- prometheus.MustNewConstMetric(
		c.metrics.topicPartitions,
		prometheus.GaugeValue, float64(partitions.Len()),
		topic.Name,
	)

	wg := sync.WaitGroup{}
	wg.Add(len(partitions))
	for _, partition := range partitions {
		go func(cz chan<- prometheus.Metric, t *kazoo.Topic, p *kazoo.Partition) {
			c.partitionMetrics(cz, t, p)
			wg.Done()
		}(ch, topic, partition)
	}
	wg.Wait()

}

// called from topicMetrics() to extract per partition metrics
func (c *collector) partitionMetrics(ch chan<- prometheus.Metric, topic *kazoo.Topic, partition *kazoo.Partition) {
	// kafka_topic_partition_replica_count{topic="name", partition="1"} 2
	ch <- prometheus.MustNewConstMetric(
		c.metrics.partitionReplicaCount,
		prometheus.GaugeValue, float64(len(partition.Replicas)),
		topic.Name, fmt.Sprint(partition.ID),
	)

	leader, err := partition.Leader()
	if err != nil {
		msg := fmt.Sprintf("Error fetching partition leader for partition %d on topic '%s': %s", partition.ID, topic.Name, err)
		log.Errorf(msg)
		ch <- prometheus.NewInvalidMetric(c.metrics.partitionLeader, errors.New(msg))
		return
	}
	// kafka_topic_partition_leader_is_preferred{topic="name", partition="1", replica="10001"} 1
	ch <- prometheus.MustNewConstMetric(
		c.metrics.partitionLeader,
		prometheus.GaugeValue, 1,
		topic.Name, fmt.Sprint(partition.ID), fmt.Sprint(leader),
	)

	isr, err := partition.ISR()
	if err != nil {
		msg := fmt.Sprintf("Error fetching partition ISR information for partition %d on topic '%s': %s", partition.ID, topic.Name, err)
		log.Errorf(msg)
		ch <- prometheus.NewInvalidMetric(c.metrics.partitionISR, errors.New(msg))
		return
	}
	for _, replica := range partition.Replicas {
		var inSync float64
		if int32InSlice(replica, isr) {
			inSync = 1
		}
		// kafka_topic_partition_replica_in_sync{topic="name", partition="1", replica="10002"} 0
		ch <- prometheus.MustNewConstMetric(
			c.metrics.partitionISR,
			prometheus.GaugeValue, inSync,
			topic.Name, fmt.Sprint(partition.ID), fmt.Sprint(replica),
		)
	}

	preferred := partition.PreferredReplica()
	var isPreferred float64
	if leader == preferred {
		isPreferred = 1
	}
	// kafka_topic_partition_leader_is_preferred{topic="name", partition="1"} 1
	ch <- prometheus.MustNewConstMetric(
		c.metrics.partitionUsesPreferredReplica,
		prometheus.GaugeValue, isPreferred,
		topic.Name, fmt.Sprint(partition.ID),
	)
}

func (c *collector) consumerMetrics(ch chan<- prometheus.Metric, consumer *kazoo.Consumergroup) {
	offsets, err := consumer.FetchAllOffsets()
	if err != nil {
		msg := fmt.Sprintf("Error collecting offset for consumer %s: %s", consumer.Name, err)
		log.Errorf(msg)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("zookeeper_consumer_offsets_error", msg, nil, nil), err)
		return
	}

	for topicName, topicParts := range offsets {
		// Don't emit metrics for filtered topics.
		// We already logged that we're skipping the topic in Collect(),
		// so there's no need to log it for every consumer.
		if len(c.topics) > 0 && !stringInSlice(topicName, c.topics) {
			continue
		}
		for partitionID, partitionOffset := range topicParts {
			// kafka_consumers_offsets{consumer="name",topic="name",partition="num"} 1234567890
			ch <- prometheus.MustNewConstMetric(
				c.metrics.consumersOffsets,
				prometheus.CounterValue, float64(partitionOffset),
				consumer.Name, topicName, fmt.Sprint(partitionID),
			)
		}
	}
}
