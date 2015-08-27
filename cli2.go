package main

import (
	"os"
	"redis-cluster/cluster.v2"
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

	redisCluster.Bootstrap()
}
