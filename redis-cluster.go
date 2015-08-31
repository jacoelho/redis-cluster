package main

import (
	"fmt"
	"os"
	"redis-cluster/cluster.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/codegangsta/cli"
)

func AssembleCluster(tag string, size int, port string) error {
	metadata := ec2metadata.New(&ec2metadata.Config{})

	region, err := metadata.Region()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	svc := ec2.New(&aws.Config{Region: aws.String(region)})

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
					aws.String(tag),
				},
			},
		},
	}

	// Call the DescribeInstances Operation
	resp, err := svc.DescribeInstances(params)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	servers := make([]string, 0)
	for idx, _ := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			servers = append(servers, fmt.Sprintf("%s:%s,%s", *inst.PrivateIpAddress, port, *inst.Placement.AvailabilityZone))
		}
	}

	if len(servers) < 3 {
		fmt.Println("invalid cluster size")
		os.Exit(1)
	}

	fmt.Println("server list:", servers)
	redisCluster := cluster.NewCluster(servers)
	if redisCluster == nil {
		fmt.Println("error creating cluster")
		os.Exit(1)
	}

	err = redisCluster.Bootstrap(size)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "redis-cluster"
	app.Usage = "auto redis-cluster"
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "masters",
			Value: 3,
			Usage: "number of masters",
		},
		cli.StringFlag{
			Name:  "tag",
			Value: "redis",
			Usage: "aws tag to use",
		},
		cli.StringFlag{
			Name:  "port",
			Value: "6379",
			Usage: "redis port",
		},
	}

	app.Action = func(c *cli.Context) {
		cluster_size := c.Int("masters")
		tag := c.String("tag")
		port := c.String("port")

		if cluster_size < 3 {
			fmt.Println("invalid cluster size")
			os.Exit(1)
		}

		if len(tag) < 1 {
			fmt.Println("invalid flag")
			os.Exit(1)
		}

		AssembleCluster(tag, cluster_size, port)

	}

	app.Run(os.Args)

}
