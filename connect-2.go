package main

import (
	"fmt"
	"gopkg.in/redis.v3"
	//"strconv"
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

type simpleNode struct {
	id     string
	master string
	slots  bool
}

func main() {
	addrs := []string{
		"172.17.0.96:6379",
		"172.17.0.97:6379",
		"172.17.0.98:6379",
		"172.17.0.99:6379",
	}

	cluster := make(map[string]int)

	next_slave := []string{}

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

		nodes := client.ClusterNodes().Val()
		memberNodes := strings.Split(nodes, "\n")

		for _, myself := range memberNodes {
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
					next_slave = append(next_slave, server)
				} else {
					if _, ok := cluster[info.id]; !ok {
						cluster[info.id] = 0
					}
				}

				//fmt.Println(info)
				//fmt.Println(myselfInfo[1])
			}
		}
	}

	fmt.Println("cluster members", cluster)
	fmt.Println("possible slaves", next_slave)

	if len(next_slave) > 0 {
		for _, server := range next_slave {
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
