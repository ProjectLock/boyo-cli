package webserver

import "fmt"

const (
	Name        = "webserver"
	Description = "Single EC2 instance inside a public VPC with Security Group"
)

func Describe() string {
	return Describe()
}

func Deploy(region string) error {
	fmt.Printf("🚀 Deploying EC2 Webserver to AWS region %s...\n", region)
	return nil
}
