package cluster

import (
	"fmt"
	"gopkg.in/redis.v3"
	"strings"
)

const CLUSTER_HASH_SLOTS = 16383
const CLUSTER_QUORUM = 3

type Cluster struct {
	addrs   []string
	size    int
	clients map[int]*redis.Client
}

type ClusterNode struct {
	id          string
	addr        string
	flags       []string
	master      string
	pingSent    string
	pongRecv    string
	configEpoch string
	linkState   string
	slot        []string
}

func parseNode(line string) *ClusterNode {
	fields := strings.Split(line, " ")

	if len(fields) < 7 {
		return nil
	}

	node := &ClusterNode{
		id:          fields[0],
		addr:        fields[1],
		flags:       strings.Split(fields[2], ","),
		master:      fields[3],
		pingSent:    fields[4],
		pongRecv:    fields[5],
		configEpoch: fields[6],
		linkState:   fields[7],
		slot:        fields[8:],
	}
	return node
}

func NewCluster(servers []string) *Cluster {
	newCluster := &Cluster{
		addrs:   servers,
		size:    len(servers),
		clients: make(map[int]*redis.Client, len(servers)),
	}

	if newCluster.size < CLUSTER_QUORUM {
		return nil
	}

	for idx, server := range newCluster.addrs {
		client := redis.NewClient(
			&redis.Options{
				Addr:     server,
				Password: "",
			},
		)

		err := client.Ping().Err()
		if err != nil {
			return nil
		}

		newCluster.clients[idx] = client
	}

	return newCluster
}

func StartCluster(redisCluster *Cluster) error {
	return nil
}

func MeetCluster(redisCluster *Cluster) error {
	for _, client := range redisCluster.clients {
		connectedNodes := CheckKnownNodes(client)

		for _, neighbour := range redisCluster.addrs {
			if _, ok := connectedNodes[neighbour]; !ok {
				ip := strings.Split(neighbour, ":")[0]
				port := strings.Split(neighbour, ":")[1]
				err := client.ClusterMeet(ip, port).Err()
				if err != nil {
					return err
				}
				fmt.Println(neighbour, "handshake sent")
			} else {
				fmt.Println(neighbour, "is already known")
			}
		}
	}
	return nil
}

func StopCluster(redisCluster *Cluster) error {
	for _, client := range redisCluster.clients {
		if err := client.Close(); err != nil {
			return err
		}
	}
	return nil
}

func CheckKnownNodes(client *redis.Client) map[string]*ClusterNode {
	result := make(map[string]*ClusterNode)
	nodes := client.ClusterNodes().Val()
	knownNodes := strings.Split(nodes, "\n")

	for _, line := range knownNodes {
		node := parseNode(line)

		if node != nil {
			result[node.addr] = node
		}
	}
	return result
}

func CheckCluster(redisCluster *Cluster) error {
	for _, client := range redisCluster.clients {
		fmt.Println("node ->>>", CheckKnownNodes(client))
	}

	return nil
}
