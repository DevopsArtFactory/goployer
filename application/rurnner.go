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

		deployers = append(deployers, _get_deployer(r.Logger, stack, r.Builder.AwsConfig))
	}

	// Deploy
	for _, deployer := range deployers {
		deployer.Deploy(r.Builder.Config)
	}

	// healthcheck
	_do_healthchecking(deployers, r.Builder.Config)

}

func _get_deployer(logger *Logger.Logger, stack Stack, awsConfig AWSConfig) DeployManager {
	deployer := _NewBlueGrean(
		stack.ReplacementType,
		logger,
		awsConfig,
		stack,
	)

	return deployer
}

func _do_healthchecking(deployers []DeployManager, config Config) {
	healthyStackList := []string{}
	healthy := false
	for ! healthy {
		for _, deployer := range deployers {
			if IsStringInArray(deployer.GetStackName(), healthyStackList) {
				continue
			}
			ret := deployer.Healthchecking(config)

			if ret {
				healthyStackList = append(healthyStackList, deployer.GetStackName())
			}
		}

		if len(healthyStackList) == len(deployers) {
			healthy = true
		} else {
			time.Sleep(POLLING_SLEEP_TIME)
		}
	}
}
