package cluster

import (
	"fmt"
	"gopkg.in/redis.v3"
	"sort"
	"strings"
)

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
	slots       []string
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
		slots:       fields[8:],
	}
	return node
}

func GenerateClusterSlots(clusterSize int) [][]int {
	// lets slot it
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

		if first == last {
			result[i] = []int{first}
		} else {
			result[i] = []int{first, last}
		}
	}

	return result
}

// implement a binary search?
func contains(slice []string, value string) bool {
	for _, i := range slice {
		if i == value {
			return true
		}
	}
	return false
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

func AddSlave(addr string, masterId string) error {
	client := redis.NewClient(
		&redis.Options{
			Addr:     addr,
			Password: "",
		},
	)

	err := client.ClusterReplicate(masterId).Err()
	if err != nil {
		return err
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

func CheckCluster(redisCluster *Cluster) (unassigned []string, masterCount map[string]int) {
	unassigned = make([]string, 0)
	masterCount = make(map[string]int)

	client := redisCluster.clients[0]
	nodes := CheckKnownNodes(client)

	for _, nodeInfo := range nodes {
		// ignore failed and handshake
		if contains(nodeInfo.flags, "handshake") || contains(nodeInfo.flags, "fail") {
			continue
		}

		// slave node - register master
		if nodeInfo.master != "-" {
			masterCount[nodeInfo.master] += 1
		} else {
			// nodes with master status and without slots assigned
			if len(nodeInfo.slots) == 0 {
				unassigned = append(unassigned, nodeInfo.addr)
			} else {
				// nodes with master status and slots assigned
				if _, ok := masterCount[nodeInfo.id]; !ok {
					masterCount[nodeInfo.id] = 0
				}
			}
		}
	}
	return
}

func AssignSlaves(unassigned []string, masterCount map[string]int) {
	for _, item := range unassigned {
		tmp := convertToPairList(masterCount)
		sort.Sort(tmp)

		fmt.Println("adding", item, "as slave of", tmp[0].Key)
		AddSlave(item, tmp[0].Key)
		masterCount[tmp[0].Key] += 1
	}
}

func AssignClusterSlots(unassigned []string, clusterSize int) error {
	slots := GenerateClusterSlots(clusterSize)

	for idx, server := range unassigned[:clusterSize] {
		client := redis.NewClient(
			&redis.Options{
				Addr:     server,
				Password: "",
			},
		)

		if len(slots[idx]) > 1 {
			err := client.ClusterAddSlotsRange(slots[idx][0], slots[idx][1]).Err()
			if err != nil {
				return err
			} else {
				err := client.ClusterAddSlots(slots[idx][0]).Err()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
