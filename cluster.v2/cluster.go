package cluster

import (
	"errors"
	"fmt"
	"gopkg.in/redis.v3"
	"sort"
	"strconv"
	"strings"
)

const CLUSTER_HASH_SLOTS = 16383 // 16384 with 0
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

type ClusterTopology struct {
	masters    map[string]int
	candidates map[string]*redis.Client
	servers    map[string]string
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

// A data structure to hold key/value pairs
type Pair struct {
	Key   string
	Value int
}

// A slice of pairs that implements sort.Interface to sort by values
type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }

func convertToPairList(m map[string]int) PairList {
	result := make(PairList, len(m))
	i := 0
	for k, v := range m {
		result[i] = Pair{k, v}
		i++
	}
	return result
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

func (cluster *Cluster) GetClient(address string) *redis.Client {
	for _, item := range cluster.Cluster_members {
		if item.address == address {
			return item.client
		}
	}
	return nil
}

func (cluster *Cluster) GetTopology() *ClusterTopology {
	if len(cluster.Cluster_members) < CLUSTER_QUORUM {
		return nil
	}

	client := cluster.Cluster_members[0].client
	nodes := GetNodes(client)
	result := &ClusterTopology{
		masters:    make(map[string]int),
		candidates: make(map[string]*redis.Client),
		servers:    make(map[string]string),
	}

	for _, node := range nodes {
		// set pod
		for _, item := range cluster.Cluster_members {
			if item.address == node.addr {
				result.servers[node.id] = item.pod
			}
		}

		if node.master != "-" {
			result.masters[node.master] += 1
		} else if node.master == "-" && len(node.slots) > 0 {
			if _, ok := result.masters[node.id]; !ok {
				result.masters[node.id] = 0
			}
		} else {
			result.candidates[node.id] = cluster.GetClient(node.addr)
		}
	}
	return result
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
		if len(strings.Split(address, ",")) != 2 {
			continue
		}
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

func (cluster *Cluster) AssignSlaves() error {
	result := cluster.GetTopology()

	if result == nil || len(result.masters) < CLUSTER_QUORUM {
		return errors.New("cannt assign slaves without masters")
	}

	masters := convertToPairList(result.masters)

	for slave, client := range result.candidates {
		sort.Sort(masters)
		assigned := false

		for idx, master := range masters {
			if result.servers[master.Key] != result.servers[slave] {
				err := client.ClusterReplicate(master.Key).Err()
				if err != nil {
					return err
				}
				masters[idx].Value += 1
				assigned = true
				break
			}
		}

		if !assigned {
			err := client.ClusterReplicate(masters[0].Key).Err()
			if err != nil {
				return err
			}
			masters[0].Value += 1
		}
	}
	return nil
}

func (cluster *Cluster) Bootstrap(size int) error {
	if cluster.State == REDIS_FAIL && cluster.Slots_assigned == 0 {
		cluster.AssignMasters(size)
		err := cluster.AssignSlaves()
		if err != nil {
			return err
		}
	} else if cluster.State == REDIS_OK {
		err := cluster.AssignSlaves()
		if err != nil {
			return err
		}
	} else {
		fmt.Println("cluster need manual repair")
	}

	return nil
}
