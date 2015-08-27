package main

import (
	"fmt"
	"math/rand"
	"os"
	"redis-cluster/cluster.v2"
)

func GetAZ() string {
	values := []string{"AZ1", "AZ2", "AZ3", "AZ4", "AZ5", "AZ6", "AZ7", "AZ8"}
	return values[rand.Intn(8)]
}

func main() {
	args := os.Args[1:]
	servers := make([]string, len(args))

	for idx, arg := range args {
		servers[idx] = fmt.Sprintf("%s:6379,%s", arg, GetAZ())
	}
	redisCluster := cluster.NewCluster(servers)
	if redisCluster == nil {
		panic("error creating cluster")
	}

	err := redisCluster.Bootstrap(4)
	if err != nil {
		panic(err)
	}

}
