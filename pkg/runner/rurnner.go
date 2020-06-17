package runner

import (
	"errors"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/deployer"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"runtime"
	"time"

	"os"
)

type Runner struct {
	Logger  *Logger.Logger
	Builder builder.Builder
	Slacker tool.Slack
}

//Start function is the starting point of all processes.
func Start() error  {
	// Check OS first
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return errors.New("you cannot run from local command.")
	}

	// Create new builder
	builder, err := builder.NewBuilder()
	if err != nil {
		return err
	}

	// Check validation of configurations
	if err := builder.CheckValidation(); err != nil {
		return err
	}

	// run with runner
	return withRunner(builder, func() error {
		// These are post actions after deployment
		return nil
	})
}


//withRunner creates runner and runs the deployment process
func withRunner(builder builder.Builder, postAction func() error ) error {
	//Prepare runnger
	runner := NewRunner(builder)
	runner.LogFormatting()
	if err := runner.Run(); err != nil {
		return err
	}

	return postAction()
}

func NewRunner(builder builder.Builder) Runner {
	return Runner{
		Logger:  Logger.New(),
		Builder: builder,
		Slacker: tool.NewSlackClient(),
	}
}

// Set log format
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
	r.Logger.Info("Beginning deployment: ", r.Builder.AwsConfig.Name)
	r.Slacker.SendSimpleMessage(r.Builder.MakeSummary(), r.Builder.Config.Env)

	//Prepare deployers
	deployers := []deployer.DeployManager{}
	for _, stack := range r.Builder.Stacks {
		// If target stack is passed from command, then
		// Skip other stacks
		if r.Builder.Config.Stack != "" && stack.Stack != r.Builder.Config.Stack {
			Logger.Info("Skipping this stack, stack=%s", stack.Stack)
			continue
		}
		d := getDeployer(r.Logger, stack, r.Builder.AwsConfig, r.Slacker)
		deployers = append(deployers, d)
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
func getDeployer(logger *Logger.Logger, stack builder.Stack, awsConfig builder.AWSConfig, slack tool.Slack) deployer.DeployManager {
	deployer := deployer.NewBlueGrean(
		stack.ReplacementType,
		logger,
		awsConfig,
		stack,
	)

	deployer.Slack = slack

	return deployer
}

// doHealthchecking checks if newly deployed autoscaling group is healthy
func doHealthchecking(deployers []deployer.DeployManager, config builder.Config) {
	healthyStackList := []string{}
	healthy := false

	ch := make(chan map[string]bool)

	for ! healthy {
		count := 0

		tool.CheckTimeout(config.StartTimestamp, config.Timeout)

		for _, deployer := range deployers {
			if tool.IsStringInArray(deployer.GetStackName(), healthyStackList) {
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
			time.Sleep(tool.POLLING_SLEEP_TIME)
		}
	}
}

// cleanChecking cleans old autoscaling groups
func cleanChecking(deployers []deployer.DeployManager, config builder.Config) {
	doneStackList := []string{}
	done := false

	ch := make(chan map[string]bool)

	for ! done {
		count := 0

		for _, deployer := range deployers {
			if tool.IsStringInArray(deployer.GetStackName(), doneStackList) {
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
			time.Sleep(tool.POLLING_SLEEP_TIME)
		}
	}
}


