package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	// Create an EC2 service object in the "us-west-2" region
	// Note that you can also configure your region globally by
	// exporting the AWS_REGION environment variable
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

	fmt.Println(resp)

	var privateIpAddress []string
	// resp has all of the response data, pull out instance IDs:
	fmt.Println("> Number of reservation sets: ", len(resp.Reservations))
	for idx, _ := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			privateIpAddress = append(privateIpAddress, *inst.PrivateIpAddress)
		}
	}

	fmt.Println(strings.Join(privateIpAddress, ","))
}
