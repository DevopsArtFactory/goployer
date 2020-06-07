package application

import (
	Logger "github.com/sirupsen/logrus"
	"time"

	"os"
)

type Runner struct {
	Logger 	*Logger.Logger
	Builder Builder
	Slacker Slack
}

//WithRunner creates runner and runs the deployment process
func WithRunner(builder Builder, postAction func() error ) error {
	// Print the summary
	builder.PrintSummary()

	//Prepare runnger
	runner := NewRunner(builder)
	runner.LogFormatting()
	if err := runner.Run(); err != nil {
		return err
	}

	return postAction()
}

func NewRunner(builder Builder) Runner {
	return Runner{
		Logger:  Logger.New(),
		Builder: builder,
		Slacker: NewSlackClient(),
	}
}

func (r Runner) LogFormatting()  {
	//logger.SetFormatter(&Logger.JSONFormatter{})
	r.Logger.SetOutput(os.Stdout)
	r.Logger.SetLevel(Logger.InfoLevel)

	r.Logger.Info("Warm up before starting deployment")
}

// Run executes all required steps for deployments
func (r Runner) Run() error  {
	defer func() {
		if err := recover(); err != nil {
			Logger.Error(err)
			os.Exit(1)
		}
	}()

	//Send Beginning Message
	r.Logger.Info("Beginning deploy for application: "+r.Builder.AwsConfig.Name)

	//Prepare deployers
	deployers := []DeployManager{}
	for _, stack := range r.Builder.Stacks {
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

	// Attach scaling policy
	for _, deployer := range deployers {
		deployer.FinishAdditionalWork(r.Builder.Config)
	}

	// Clear previous Version
	for _, deployer := range deployers {
		deployer.CleanPreviousVersion(r.Builder.Config)
	}

	// Checking all previous version before delete asg
	cleanChecking(deployers, r.Builder.Config)

	return nil
}

//Generate new deployer
func getDeployer(logger *Logger.Logger, stack Stack, awsConfig AWSConfig) DeployManager {
	deployer := NewBlueGrean(
		stack.ReplacementType,
		logger,
		awsConfig,
		stack,
	)

	return deployer
}

// doHealthchecking checks if newly deployed autoscaling group is healthy
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
				ch <- deployer.HealthChecking(config)
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

// cleanChecking cleans old autoscaling groups
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
					Logger.Info("Finished stack : ", stack)
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


