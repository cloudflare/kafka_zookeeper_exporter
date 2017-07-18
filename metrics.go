package main

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	kazoo "github.com/wvanbergen/kazoo-go"
)

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
	ch <- prometheus.MustNewConstMetric(c.metrics.topicPartitions, prometheus.GaugeValue, float64(partitions.Len()), topic.Name)

	for _, partition := range partitions {
		c.partitionMetrics(ch, topic, partition)
	}

}

// called from topicMetrics() to extract per partition metrics
func (c *collector) partitionMetrics(ch chan<- prometheus.Metric, topic *kazoo.Topic, partition *kazoo.Partition) {
	// kafka_topic_partition_replica_count{topic="name"} 2
	ch <- prometheus.MustNewConstMetric(c.metrics.partitionReplicaCount, prometheus.GaugeValue, float64(len(partition.Replicas)), topic.Name)

	leader, err := partition.Leader()
	if err != nil {
		msg := fmt.Sprintf("Error fetching partition leader for partition %d on topic '%s': %s", partition.ID, topic.Name, err)
		log.Errorf(msg)
		ch <- prometheus.NewInvalidMetric(c.metrics.partitionLeader, errors.New(msg))
		return
	}
	// kafka_topic_partition_leader_is_preferred{topic="name", partition="1", replica="10001"} 1
	ch <- prometheus.MustNewConstMetric(c.metrics.partitionLeader, prometheus.GaugeValue, 1, topic.Name, fmt.Sprint(partition.ID), fmt.Sprint(leader))

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
		ch <- prometheus.MustNewConstMetric(c.metrics.partitionISR, prometheus.GaugeValue, inSync, topic.Name, fmt.Sprint(partition.ID), fmt.Sprint(replica))
	}

	preferred := partition.PreferredReplica()
	var isPreferred float64
	if leader == preferred {
		isPreferred = 1
	}
	// kafka_topic_partition_leader_is_preferred{topic="name", partition="1"} 1
	ch <- prometheus.MustNewConstMetric(c.metrics.partitionUsesPreferredReplica, prometheus.GaugeValue, isPreferred, topic.Name, fmt.Sprint(partition.ID))
}
