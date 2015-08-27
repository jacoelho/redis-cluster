package main

import (
	"fmt"
	"math/rand"
	"os"
	"redis-cluster/cluster.v2"
)

func GetAZ() string {
	values := []string{"AZ1", "AZ2", "AZ3"}
	return values[rand.Intn(2)]
}

func main() {
	args := os.Args[1:]
	servers := make([]string, len(args))

	for idx, arg := range args {
		servers[idx] = fmt.Sprintf("%s:6379,%s", arg, GetAZ())
	}
	fmt.Println(servers)
	redisCluster := cluster.NewCluster(servers)
	if redisCluster == nil {
		panic("error creating cluster")
	}

	redisCluster.Bootstrap()

	for _, item := range redisCluster.Cluster_members {
		fmt.Println(item)
	}
}
