package staticsite

import "fmt"

const (
	Name        = "static-site"
	Description = "S3 Bucket + CloudFront distribution with Origin Access Control"
)

func Describe() string {
	return Description
}

func Deploy(region string) error {
	fmt.Printf("Deploying Static Site to AWS region %s...\n", region)
	return nil
}
