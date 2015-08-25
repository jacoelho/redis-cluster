package main

import (
	"fmt"
	"gopkg.in/redis.v3"
	"strconv"
	"strings"
)

const CLUSTER_HASH_SLOTS = 16383
const CLUSTER_MASTERS = 3

type redisNode struct {
	id          string
	addr        string
	flags       []string
	master      string
	pingSent    string
	pongRecv    string
	configEpoch string
	linkState   string
	slot        [][]int
}

type cluster struct {
	nodeIds []string
	masters []string
	slaves  []string
	clients map[string]*redis.Client
}

func main() {
	addrs := []string{
		"172.17.0.89",
		"172.17.0.91",
		"172.17.0.92",
		"172.17.0.93",
	}

	for _, server := range addrs {
		client := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{server + ":6379"},
			Password: "",
		})

		pong, err := client.Ping().Result()
		fmt.Println(pong, err)
		info := client.ClusterInfo().Val()

		if strings.Contains(info, "cluster_state:fail") {
			fmt.Println("cluster down")
		} else if strings.Contains(info, "cluster_state:ok") {
			fmt.Println("cluster up")
		}

		nodes := client.ClusterNodes().Val()
		n_nodes := strings.Split(nodes, "\n")
		for _, val := range n_nodes {
			node := strings.Split(val, " ")

			if len(node) > 7 {
				bla := &redisNode{
					id:          node[0],
					addr:        node[1],
					flags:       strings.Split(node[2], ","),
					master:      node[3],
					pingSent:    node[4],
					pongRecv:    node[5],
					configEpoch: node[6],
					linkState:   node[7],
					slot:        make([][]int, len(node[8:])),
				}

				for idx, item := range node[8:] {
					value := strings.Split(item, "-")
					first, _ := strconv.Atoi(value[0])

					if len(value) > 1 {
						last, _ := strconv.Atoi(value[1])
						bla.slot[idx] = []int{first, last}
					} else {
						bla.slot[idx] = []int{first}
					}
				}

				fmt.Println(bla)

			}
		}

		return

		// lets meet!!
		for _, meet := range addrs {
			err := client.ClusterMeet(meet, "6379").Err()
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	// lets slot it
	step := CLUSTER_HASH_SLOTS / CLUSTER_MASTERS
	result := make([][]int, CLUSTER_MASTERS)
	for i := 0; i < CLUSTER_MASTERS; i++ {

		first := i * step
		last := (i + 1) * step

		if i > 0 {
			first += 1
		}

		if (i + 1) == CLUSTER_MASTERS {
			last = CLUSTER_HASH_SLOTS
		}

		if first == last {
			result[i] = []int{first}
		} else {
			result[i] = []int{first, last}
		}
	}
	fmt.Println(result)

	// let add slots
	for idx, server := range addrs {
		client := redis.NewClusterClient(
			&redis.ClusterOptions{
				Addrs:    []string{server + ":6379"},
				Password: "",
			},
		)

		if len(result[idx]) > 1 {
			err := client.ClusterAddSlotsRange(result[idx][0], result[idx][1])
			if err != nil {
				fmt.Println(err)
			} else {
				err := client.ClusterAddSlots(result[idx][0])
				if err != nil {
					fmt.Println(err)
				}
			}

		}
	}
}
