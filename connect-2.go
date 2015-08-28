package cenas

import (
	"fmt"
	"gopkg.in/redis.v3"
	//"strconv"
	"sort"
	"strings"
)

const CLUSTER_HASH_SLOTS = 16383
const CLUSTER_MASTERS = 3

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

type simpleNode struct {
	id     string
	master string
	slots  bool
}

func generateSlots(clusterSize int) [][]int {
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

func attachSlotsToServer(serversList []string, slotList [][]int) {
	numberOfMasters := len(slotList)

	for idx, server := range serversList[:numberOfMasters] {
		client := redis.NewClient(
			&redis.Options{
				Addr:     server,
				Password: "",
			},
		)

		if len(slotList[idx]) > 1 {
			err := client.ClusterAddSlotsRange(slotList[idx][0], slotList[idx][1]).Err()
			if err != nil {
				fmt.Println(err)
			} else {
				err := client.ClusterAddSlots(slotList[idx][0]).Err()
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}

func main() {
	addrs := []string{
		"172.17.0.113:6379",
		"172.17.0.105:6379",
		"172.17.0.106:6379",
		"172.17.0.107:6379",
		"172.17.0.108:6379",
	}

	cluster := make(map[string]int)

	slaveCandidates := []string{}

	if len(addrs) < 3 {
		fmt.Println("insufficient cluster members")
		return
	}

	for _, server := range addrs {
		client := redis.NewClient(
			&redis.Options{
				Addr:     server,
				Password: "",
			},
		)

		for _, neighbourMember := range addrs {
			if server == neighbourMember {
				continue
			}

			addr := strings.Split(neighbourMember, ":")[0]
			port := strings.Split(neighbourMember, ":")[1]

			err := client.ClusterMeet(addr, port).Err()
			if err != nil {
				fmt.Println(err)
			}
		}

		nodes := client.ClusterNodes().Val()
		memberNodes := strings.Split(nodes, "\n")

		for _, myself := range memberNodes {
			if strings.Contains(myself, "slave,fail") {

			}

			if strings.Contains(myself, "myself") {
				myselfInfo := strings.Split(myself, " ")

				info := &simpleNode{
					id:     myselfInfo[0],
					master: myselfInfo[3],
					slots:  false,
				}

				if len(myselfInfo[8:]) > 0 {
					info.slots = true
				}

				// increment slave count
				if info.master != "-" {
					cluster[info.master] += 1
				}

				if info.master == "-" && (!info.slots) {
					slaveCandidates = append(slaveCandidates, server)
				} else {
					if _, ok := cluster[info.id]; !ok {
						cluster[info.id] = 0
					}
				}
			}
		}
	}

	fmt.Println("cluster members", cluster)
	fmt.Println("possible slaves", slaveCandidates)

	if len(cluster) == 0 {
		redisSlots := generateSlots(3)

		// add masters
		attachSlotsToServer(addrs, redisSlots)

		// calculate slaves

		return
	}

	if len(slaveCandidates) > 0 {
		for _, server := range slaveCandidates {
			client := redis.NewClient(
				&redis.Options{
					Addr:     server,
					Password: "",
				},
			)

			sortableMasters := make(PairList, len(cluster))
			i := 0
			for k, v := range cluster {
				sortableMasters[i] = Pair{k, v}
				i++
			}

			sort.Sort(sortableMasters)

			fmt.Println("sorted masters: ", sortableMasters)
			fmt.Println("adding slave ", server, " of ", sortableMasters[0].Key)

			err := client.ClusterReplicate(sortableMasters[0].Key).Err()
			if err != nil {
				fmt.Println(err)
			}
		}
	}

}
