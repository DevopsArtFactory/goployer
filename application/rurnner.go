package application

import (
	"fmt"
	Logger "github.com/sirupsen/logrus"
	"time"

	"os"
)

type Runner struct {
	Logger 	*Logger.Logger
	Builder   Builder
	Slacker Slack
}

func NewRunner(builder Builder) Runner {
	return Runner{
		Logger:  Logger.New(),
		Builder: builder,
		Slacker: NewSlackClient(),
	}
}

func (r Runner) WarmUp()  {
	//logger.SetFormatter(&Logger.JSONFormatter{})
	r.Logger.SetOutput(os.Stdout)
	r.Logger.SetLevel(Logger.InfoLevel)

	r.Logger.Info("Warm up before starting deployment")
}

// Run
func (r Runner) Run()  {

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	//Send Beginning Message

	r.Logger.Info("Beginning deploy for application: "+r.Builder.AwsConfig.Name)

	deployers := []DeployManager{}
	for _, stack := range r.Builder.Stacks.Stacks {
		// If target stack is passed from command, then
		// Skip other stacks
		if r.Builder.Config.Stack != "" && stack.Stack != r.Builder.Config.Stack {
			Logger.Info("Skipping this stack, stack=%s", stack.Stack)
			continue
		}

		deployers = append(deployers, getDeployer(r.Logger, stack, r.Builder.AwsConfig))
	}

	// Deploy
	for _, deployer := range deployers {
		deployer.Deploy(r.Builder.Config)
	}

	// healthcheck
	doHealthchecking(deployers, r.Builder.Config)

	// Clear previous Version
	for _, deployer := range deployers {
		deployer.CleanPreviousVersion()
	}

	// Checking all previous version before delete asg
	cleanChecking(deployers, r.Builder.Config)
}

//Generate new deployer
func getDeployer(logger *Logger.Logger, stack Stack, awsConfig AWSConfig) DeployManager {
	deployer := _NewBlueGrean(
		stack.ReplacementType,
		logger,
		awsConfig,
		stack,
	)

	return deployer
}


func doHealthchecking(deployers []DeployManager, config Config) {
	healthyStackList := []string{}
	healthy := false

	ch := make(chan map[string]bool)

	for ! healthy {
		count := 0

		checkTimeout(config.StartTimestamp, config.Timeout)

		for _, deployer := range deployers {
			if IsStringInArray(deployer.GetStackName(), healthyStackList) {
				continue
			}

			count += 1

			//Start healthcheck thread
			go func() {
				ch <- deployer.Healthchecking(config)
			}()
		}

		for count > 0 {
			ret := <- ch
			for stack, fin := range ret {
				if fin {
					healthyStackList = append(healthyStackList, stack)
				}
			}
			count -= 1
		}

		if len(healthyStackList) == len(deployers) {
			Logger.Info("All stacks are healthy")
			healthy = true
		} else {
			Logger.Info("All stacks are not healthy... Please waiting to be deployed...")
			time.Sleep(POLLING_SLEEP_TIME)
		}
	}
}


func cleanChecking(deployers []DeployManager, config Config) {
	doneStackList := []string{}
	done := false

	ch := make(chan map[string]bool)

	for ! done {
		count := 0

		for _, deployer := range deployers {
			if IsStringInArray(deployer.GetStackName(), doneStackList) {
				continue
			}

			count += 1

			//Start terminateChecking thread
			go func() {
				ch <- deployer.TerminateChecking(config)
			}()
		}

		for count > 0 {
			ret := <- ch
			for stack, fin := range ret {
				if fin {
					doneStackList = append(doneStackList, stack)
				}
			}
			count -= 1
		}

		if len(doneStackList) == len(deployers) {
			Logger.Info("All stacks are terminated!!")
			done = true
		} else {
			Logger.Info("All stacks are not ready to be terminated... Please waiting...")
			time.Sleep(POLLING_SLEEP_TIME)
		}
	}
}


