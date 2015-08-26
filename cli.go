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

	fmt.Print("-->", count)

	if len(count) > 0 && len(count) < 3 {
		panic("something is wrong")
	}

	fmt.Println("unassigned", unassigned)
	if len(count) == 0 {
		err := cluster.AssignClusterSlots(unassigned, 4)
		if err != nil {
			fmt.Println(err)
		}
	}

	unassigned, count = cluster.CheckCluster(redisCluster)

	//cluster.AssignSlaves(unassigned, count)

	fmt.Println(cluster.CheckCluster(redisCluster))

}
