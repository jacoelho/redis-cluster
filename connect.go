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
		"172.17.0.96",
		"172.17.0.97",
		"172.17.0.98",
		"172.17.0.99",
	}

	if len(addrs) < 3 {
		fmt.Println("insufficient cluster members")
	}

	cluster := make(map[string]map[string]*redisNode, len(addrs))

	for _, server := range addrs {
		client := redis.NewClusterClient(
			&redis.ClusterOptions{
				Addrs:    []string{server + ":6379"},
				Password: "",
			},
		)

		info := client.ClusterInfo().Val()

		if strings.Contains(info, "cluster_state:fail") {
			fmt.Println("cluster down")
		} else if strings.Contains(info, "cluster_state:ok") {
			fmt.Println("cluster up")
		}

		nodes := client.ClusterNodes().Val()
		n_nodes := strings.Split(nodes, "\n")

		fmt.Println("checking server", server)
		if _, ok := cluster[server]; !ok {
			cluster[server] = make(map[string]*redisNode, len(n_nodes))
		}

		for _, val := range n_nodes {
			field := strings.Split(val, " ")

			if len(field) > 7 {
				bla := &redisNode{
					id:          field[0],
					addr:        field[1],
					flags:       strings.Split(field[2], ","),
					master:      field[3],
					pingSent:    field[4],
					pongRecv:    field[5],
					configEpoch: field[6],
					linkState:   field[7],
					slot:        make([][]int, len(field[8:])),
				}

				for idx, item := range field[8:] {
					value := strings.Split(item, "-")

					fmt.Println("debug", value)

					first, _ := strconv.Atoi(value[0])

					if len(value) > 1 {
						last, _ := strconv.Atoi(value[1])
						bla.slot[idx] = []int{first, last}
					} else {
						bla.slot[idx] = []int{first}
					}
				}

				for _, flag := range bla.flags {
					if flag == "myself" {
						fmt.Println(server, bla.slot)
					}
				}

				cluster[server][bla.addr] = bla

			}

		}

		// lets meet!!
		for _, meet := range addrs {

			_, ok := cluster[server][meet+":6379"]
			if ok {
				fmt.Println("node already known")
			} else {
				fmt.Println("adding new node")

				err := client.ClusterMeet(meet, "6379").Err()
				if err != nil {
					fmt.Println(err)
				}
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
	for idx, server := range addrs[:CLUSTER_MASTERS] {
		client := redis.NewClusterClient(
			&redis.ClusterOptions{
				Addrs:    []string{server + ":6379"},
				Password: "",
			},
		)

		if len(result[idx]) > 1 {
			err := client.ClusterAddSlotsRange(result[idx][0], result[idx][1]).Err()
			if err != nil {
				fmt.Println(err)
			} else {
				err := client.ClusterAddSlots(result[idx][0]).Err()
				if err != nil {
					fmt.Println(err)
				}
			}

		}
	}

	fmt.Println(cluster)

	for key, value := range cluster {
		fmt.Println(key)
		for _, new_value := range value {
			fmt.Println("-->", new_value.flags, new_value.slot)
		}
	}
}
