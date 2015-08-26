package main

import (
	"fmt"
	"redis-cluster/cluster"
	"time"
)

func main() {
	redisCluster := cluster.NewCluster([]string{"172.17.0.120:6379", "172.17.0.119:6379", "172.17.0.116:6379"})

	//cluster.MeetCluster(redisCluster)

	time.Sleep(5 * 1000 * time.Millisecond)

	unassigned, count := cluster.CheckCluster(redisCluster)

	if len(count) > 0 && len(count) < 3 {
		panic("something is wrong")
	}

	if len(count) == 0 {
		cluster.AssignClusterSlots(unassigned, 4)
	}

	unassigned, count = cluster.CheckCluster(redisCluster)

	cluster.AssignSlaves(unassigned, count)

	fmt.Println(cluster.CheckCluster(redisCluster))

}
