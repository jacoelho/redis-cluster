package main

import (
	"fmt"
	"os"
	"redis-cluster/cluster"
	"time"
)

func main() {
	args := os.Args[1:]
	servers := make([]string, len(args))

	for idx, arg := range args {
		servers[idx] = arg + ":6379"
	}
	redisCluster := cluster.NewCluster(servers)
	if redisCluster == nil {
		panic("error creating cluster")
	}

	cluster.MeetCluster(redisCluster)

	time.Sleep(5 * 1000 * time.Millisecond)

	unassigned, count := cluster.CheckCluster(redisCluster)

	if len(count) > 0 && len(count) < 3 {
		panic("something is wrong")
	}

	if len(count) == 0 {
		err := cluster.AssignClusterSlots(unassigned, 3)
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(5 * 1000 * time.Millisecond)
	}

	unassigned, count = cluster.CheckCluster(redisCluster)

	cluster.AssignSlaves(unassigned, count)

	//fmt.Println(cluster.CheckCluster(redisCluster))

}
