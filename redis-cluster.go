package main

import (
	"fmt"
	"os"
	"redis-cluster/cluster.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	svc := ec2.New(&aws.Config{Region: aws.String("us-west-1")})

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
			{
				Name: aws.String("tag:role"),
				Values: []*string{
					aws.String("redis"),
				},
			},
		},
	}

	// Call the DescribeInstances Operation
	resp, err := svc.DescribeInstances(params)

	if err != nil {
		panic(err)
	}

	servers := make([]string, 0)
	for idx, _ := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			servers = append(servers, fmt.Sprintf("%s:6379,%s", *inst.PrivateIpAddress, *inst.Placement.AvailabilityZone))
		}
	}

	fmt.Println("server list:", servers)
	redisCluster := cluster.NewCluster(servers)
	if redisCluster == nil {
		fmt.Println("error creating cluster")
		os.Exit(1)
	}

	err = redisCluster.Bootstrap(4)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
