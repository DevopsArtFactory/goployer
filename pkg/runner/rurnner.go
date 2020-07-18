package runner

import (
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/collector"
	"github.com/DevopsArtFactory/goployer/pkg/deployer"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"time"

	"os"
)

type Runner struct {
	Logger    *Logger.Logger
	Builder   builder.Builder
	Collector collector.Collector
	Slacker   tool.Slack
}

var (
	logLevelMapper = map[string]Logger.Level{
		"info":  Logger.InfoLevel,
		"debug": Logger.DebugLevel,
		"warn":  Logger.WarnLevel,
		"trace": Logger.TraceLevel,
		"fatal": Logger.FatalLevel,
		"error": Logger.ErrorLevel,
	}
)

//Start function is the starting point of all processes.
func Start() error {
	// Check OS first
	//if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
	//	return errors.New("you cannot run from local command.")
	//}

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
	return withRunner(builder, func(slacker tool.Slack) error {
		// These are post actions after deployment
		slacker.SendSimpleMessage(":100: Deployment is done.", builder.Config.Env)
		return nil
	})
}

//withRunner creates runner and runs the deployment process
func withRunner(builder builder.Builder, postAction func(slacker tool.Slack) error) error {
	runner, err := NewRunner(builder)
	if err != nil {
		return err
	}
	runner.LogFormatting(builder.Config.LogLevel)
	if err := runner.Run(); err != nil {
		return err
	}

	return postAction(runner.Slacker)
}

//NewRunner creates a new runner
func NewRunner(newBuilder builder.Builder) (Runner, error) {
	m, err := builder.ParseMetricConfig(newBuilder.Config.DisableMetrics)
	if err != nil {
		return Runner{}, err
	}

	newBuilder.MetricConfig = m

	return Runner{
		Logger:    Logger.New(),
		Builder:   newBuilder,
		Collector: collector.NewCollector(m, newBuilder.Config.AssumeRole),
		Slacker:   tool.NewSlackClient(newBuilder.Config.SlackOff),
	}, nil
}

// Set log format
func (r Runner) LogFormatting(logLevel string) {
	//logger.SetFormatter(&Logger.JSONFormatter{})
	r.Logger.SetOutput(os.Stdout)
	r.Logger.SetLevel(logLevelMapper[logLevel])
}

// Run executes all required steps for deployments
func (r Runner) Run() error {
	defer func() {
		if err := recover(); err != nil {
			Logger.Error(err)
			os.Exit(1)
		}
	}()

	//Send Beginning Message
	r.Logger.Info("Beginning deployment: ", r.Builder.AwsConfig.Name)

	msg := r.Builder.MakeSummary(r.Builder.Config.Stack)
	fmt.Println(msg)
	if r.Slacker.ValidClient() {
		r.Logger.Debug("slack configuration is valid")
		err := r.Slacker.SendSimpleMessage(msg, r.Builder.Config.Env)
		if err != nil {
			r.Logger.Warn(err.Error())
			r.Slacker.SlackOff = true
		}
	} else {
		// Slack variables are not set
		r.Logger.Warnln("no slack variables exists. [ SLACK_TOKEN, SLACK_CHANNEL ]")
	}

	if r.Builder.MetricConfig.Enabled {
		Logger.Infof("Metric Measurement is enabled")

		Logger.Debugf("check if storage exists or not")
		if err := r.Collector.CheckStorage(); err != nil {
			return err
		}
	}

	Logger.Debug("create deployers for stacks")

	//Prepare deployers
	deployers := []deployer.DeployManager{}
	for _, stack := range r.Builder.Stacks {
		// If target stack is passed from command, then
		// Skip other stacks
		if r.Builder.Config.Stack != "" && stack.Stack != r.Builder.Config.Stack {
			Logger.Debugf("Skipping this stack, stack=%s", stack.Stack)
			continue
		}
		d := getDeployer(r.Logger, stack, r.Builder.AwsConfig, r.Slacker, r.Collector)
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

	// Trigger Lifecycle Callbacks
	for _, deployer := range deployers {
		deployer.TriggerLifecycleCallbacks(r.Builder.Config)
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
func getDeployer(logger *Logger.Logger, stack builder.Stack, awsConfig builder.AWSConfig, slack tool.Slack, c collector.Collector) deployer.DeployManager {
	deployer := deployer.NewBlueGrean(
		stack.ReplacementType,
		logger,
		awsConfig,
		stack,
	)

	deployer.Slack = slack
	deployer.Collector = c

	return deployer
}

// doHealthchecking checks if newly deployed autoscaling group is healthy
func doHealthchecking(deployers []deployer.DeployManager, config builder.Config) {
	healthyStackList := []string{}
	healthy := false

	ch := make(chan map[string]bool)

	for !healthy {
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
			ret := <-ch
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

	for !done {
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
			ret := <-ch
			for stack, fin := range ret {
				if fin {
					Logger.Debug("Finished stack : ", stack)
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
