package main

import (
	"fmt"
	"redis-cluster/cluster"
	"time"
)

func main() {
	redisCluster := cluster.NewCluster([]string{"172.17.0.120:6379", "172.17.0.119:6379", "172.17.0.116:6379"})

	cluster.MeetCluster(redisCluster)

	time.Sleep(5 * 1000 * time.Millisecond)

	cluster.CheckCluster(redisCluster)

	fmt.Println(redisCluster)
}
