package cluster

import (
	"errors"
	"gopkg.in/redis.v3"
	"strconv"
	"strings"

	"fmt"
)

const CLUSTER_HASH_SLOTS = 16383
const CLUSTER_QUORUM = 3
const REDIS_FAIL = false
const REDIS_OK = true

type RedisNode struct {
	id     string
	addr   string
	flags  []string
	master string
	//	pingSent    string
	//	pongRecv    string
	//	configEpoch string
	//	linkState   string
	slots []string
}

type ClusterNode struct {
	address string
	pod     string
	client  *redis.Client
}

type Cluster struct {
	State           bool
	Slots_assigned  int
	Cluster_members []*ClusterNode
}

func contains(slice []string, value string) bool {
	for _, i := range slice {
		if i == value {
			return true
		}
	}
	return false
}

func GenerateClusterSlots(clusterSize int) [][]int {
	step := CLUSTER_HASH_SLOTS / clusterSize
	result := make([][]int, clusterSize)

	for i := 0; i < clusterSize; i++ {
		first := i * step
		last := (i + 1) * step

		if i > 0 {
			first += 1
		}

		if (i + 1) == clusterSize {
			last = CLUSTER_HASH_SLOTS
		}

		// avoid issues with step size 1
		if first == last {
			result[i] = []int{first}
		} else {
			result[i] = []int{first, last}
		}
	}

	return result
}

func parseNodeOutput(line string) *RedisNode {
	fields := strings.Split(line, " ")

	if len(fields) < 7 {
		return nil
	}

	node := &RedisNode{
		id:     fields[0],
		addr:   fields[1],
		flags:  strings.Split(fields[2], ","),
		master: fields[3],
		slots:  fields[8:],
	}

	// ignore if ip address not defined
	checkAddr := strings.Split(node.addr, ":")
	if len(checkAddr[0]) == 0 {
		return nil
	}
	return node
}

func MeetNode(client *redis.Client, address string) error {
	ip := strings.Split(address, ":")[0]
	port := strings.Split(address, ":")[1]

	err := client.ClusterMeet(ip, port).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetClusterInfo(client *redis.Client) (status bool, slots int) {
	new_status := REDIS_FAIL
	used_slots := 0

	tmp, err := client.ClusterInfo().Result()

	if err != nil {
		return new_status, used_slots
	}

	result := strings.Split(tmp, "\n")

	if strings.Contains(result[0], "cluster_state:ok") {
		new_status = REDIS_OK
	}

	value := strings.Split(result[1], ":")[1]
	used_slots, _ = strconv.Atoi(strings.TrimSpace(value))

	return new_status, used_slots
}

func GetNodes(client *redis.Client) map[string]*RedisNode {
	result := make(map[string]*RedisNode)
	nodes := client.ClusterNodes().Val()
	knownNodes := strings.Split(nodes, "\n")

	for _, line := range knownNodes {
		node := parseNodeOutput(line)

		if node != nil {
			result[node.addr] = node
		}
	}
	return result
}

func NewCluster(initialList []string) *Cluster {
	cluster := &Cluster{
		State:           REDIS_OK,
		Slots_assigned:  0,
		Cluster_members: make([]*ClusterNode, 0),
	}

	for _, address := range initialList {
		ip := strings.Split(address, ",")[0]
		pod := strings.Split(address, ",")[1]

		client := redis.NewClient(
			&redis.Options{
				Addr:     ip,
				Password: "",
			},
		)

		// get known nodes
		nodes := GetNodes(client)

		// meet if needed
		for _, neighbour := range initialList {
			if _, ok := nodes[neighbour]; !ok {
				MeetNode(client, neighbour)
			}
		}

		new_state, slots := GetClusterInfo(client)

		if cluster.State != new_state {
			cluster.State = new_state
		}

		if cluster.Slots_assigned < slots {
			cluster.Slots_assigned = slots
		}

		//client.Close()
		cluster.Cluster_members = append(cluster.Cluster_members, &ClusterNode{address: ip, pod: pod, client: client})
	}

	return cluster
}

func (cluster *Cluster) AssignMasters(size int) error {
	slots := GenerateClusterSlots(size)

	if len(cluster.Cluster_members) < size {
		return errors.New("failed add master servers")
	}

	for idx, server := range cluster.Cluster_members[:size] {
		client := server.client

		// range [1000 2000] or single value [1000] [1001]
		if len(slots[idx]) > 1 {
			client.ClusterAddSlotsRange(slots[idx][0], slots[idx][1])
		} else {
			client.ClusterAddSlots(slots[idx][0])
		}
	}
	return nil
}

func (cluster *Cluster) Bootstrap() error {
	if cluster.State == REDIS_FAIL && cluster.Slots_assigned == 0 {
		fmt.Println("Assign Masters")
		cluster.AssignMasters(3)
		fmt.Println("Assign Slaves")
	} else if cluster.State == REDIS_OK {
		fmt.Println("Assign Slaves")
	} else {
		fmt.Println("cluster need manual repair")
	}

	return nil
}
