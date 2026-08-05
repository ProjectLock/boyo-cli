package main

import (
	"fmt"
	"os"

	"github.com/ProjectLock/boyo-cli/cloud/aws/templates/bastionhost"
	"github.com/ProjectLock/boyo-cli/cloud/aws/templates/staticsite"
	"github.com/ProjectLock/boyo-cli/cloud/aws/templates/webserver"

	"github.com/spf13/cobra"
)

type Template struct {
	Name        string
	Description string
	DeployFunc  func(region string) error
}

var registry = map[string]Template{
	staticsite.Name: {
		Name:        staticsite.Name,
		Description: staticsite.Description,
		DeployFunc:  staticsite.Deploy,
	},
	webserver.Name: {
		Name:        webserver.Name,
		Description: webserver.Description,
		DeployFunc:  webserver.Deploy,
	},
	bastionhost.Name: {
		Name:        bastionhost.Name,
		Description: bastionhost.Description,
		DeployFunc:  bastionhost.Deploy,
	},
}

var (
	template string
	region   string
)

func listTemplates() {
	fmt.Println("Available templates:")
	for _, t := range registry {
		fmt.Printf("  - %s: %s\n", t.Name, t.Description)
	}
}

func createInfra(templateName string, region string) {
	t, ok := registry[templateName]
	if !ok {
		fmt.Printf("Error: template '%s' not found\n", templateName)
		return
	}

	if err := t.DeployFunc(region); err != nil {
		fmt.Printf("Error deploying template '%s': %v\n", templateName, err)
		return
	}
}

func main() {
	fmt.Println("Helo, Mae Boyo yn barod!")

	var rootCMD = &cobra.Command{
		Use:   "boyo",
		Short: "Boyo is a CLI tool for cloud deployment templates.",
		Long:  `Boyo is a CLI tool for prototyping cloud infrastructure.`,
	}

	var createCMD = &cobra.Command{
		Use:   "create",
		Short: "Create cloud resources from a template.",

		Run: func(cmd *cobra.Command, args []string) {
			createInfra(template, region)
		},
	}

	createCMD.Flags().StringVarP(&template, "template", "t", "", "Template to deploy (Required)")
	createCMD.Flags().StringVarP(&region, "region", "r", "us-east-1", "AWS Region")

	createCMD.MarkFlagRequired("template")

	var listCMD = &cobra.Command{
		Use:   "list",
		Short: "List templates.",

		Run: func(cmd *cobra.Command, args []string) {
			listTemplates()
		},
	}

	rootCMD.AddCommand(createCMD)
	rootCMD.AddCommand(listCMD)

	if err := rootCMD.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
