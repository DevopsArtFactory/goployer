package main

import (
	"github.com/GwonsooLee/deployer/application"
)

func main()  {
	builder := application.NewBuilder()
	builder.CheckValidation()

	builder.PrintSummary()

	//Prepare Deployment
	runner := application.NewRunner(builder)
	runner.WarmUp()

	runner.Run()
}