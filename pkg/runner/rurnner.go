package runner

import (
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/collector"
	"github.com/DevopsArtFactory/goployer/pkg/deployer"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"strings"
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

func SetupBuilder() (builder.Builder, error) {
	// Create new builder
	builderSt, err := builder.NewBuilder(nil)
	if err != nil {
		return builder.Builder{}, err
	}

	builderSt, err = setManifestToBuilder(builderSt)
	if err != nil {
		return builder.Builder{}, err
	}

	m, err := builder.ParseMetricConfig(builderSt.Config.DisableMetrics)
	if err != nil {
		return builder.Builder{}, err
	}

	builderSt.MetricConfig = m

	return builderSt, nil
}

func ServerSetup(config builder.Config) (builder.Builder, error) {
	// Create new builder
	builderSt, err := builder.NewBuilder(&config)
	if err != nil {
		return builder.Builder{}, err
	}

	builderSt, err = setManifestToBuilder(builderSt)
	if err != nil {
		return builder.Builder{}, err
	}

	m, err := builder.ParseMetricConfig(builderSt.Config.DisableMetrics)
	if err != nil {
		return builder.Builder{}, err
	}

	builderSt.MetricConfig = m

	return builderSt, nil
}

func setManifestToBuilder(builderSt builder.Builder) (builder.Builder, error) {
	if !strings.HasPrefix(builderSt.Config.Manifest, builder.S3_PREFIX) {
		builderSt = builderSt.SetManifestConfig()
	} else {
		s := aws.BootstrapManifestService(builderSt.Config.ManifestS3Region, "")
		fileBytes, err := s.S3Service.GetManifest(filterS3Path(builderSt.Config.Manifest))
		if err != nil {
			return builder.Builder{}, err
		}
		builderSt = builderSt.SetManifestConfigWithS3(fileBytes)
	}

	return builderSt, nil
}

//Start function is the starting point of all processes.
func Start(builderSt builder.Builder) error {
	// Check validation of configurations
	if err := builderSt.CheckValidation(); err != nil {
		return err
	}

	// run with runner
	return withRunner(builderSt, func(slacker tool.Slack) error {
		// These are post actions after deployment
		slacker.SendSimpleMessage(":100: Deployment is done.", builderSt.Config.Env)

		return nil
	})
}

//withRunner creates runner and runs the deployment process
func withRunner(builderSt builder.Builder, postAction func(slacker tool.Slack) error) error {
	runner, err := NewRunner(builderSt)
	if err != nil {
		return err
	}
	runner.LogFormatting(builderSt.Config.LogLevel)

	if err := runner.Run(); err != nil {
		return err
	}

	return postAction(runner.Slacker)
}

//NewRunner creates a new runner
func NewRunner(newBuilder builder.Builder) (Runner, error) {
	return Runner{
		Logger:    Logger.New(),
		Builder:   newBuilder,
		Collector: collector.NewCollector(newBuilder.MetricConfig, newBuilder.Config.AssumeRole),
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
		r.Logger.Infof("Metric Measurement is enabled")

		r.Logger.Debugf("check if storage exists or not")
		if err := r.Collector.CheckStorage(r.Logger); err != nil {
			return err
		}
	}

	r.Logger.Debug("create deployers for stacks")

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
	if err := doHealthchecking(deployers, r.Builder.Config, r.Logger); err != nil {
		return err
	}

	// Attach scaling policy
	for _, deployer := range deployers {
		if err := deployer.FinishAdditionalWork(r.Builder.Config); err != nil {
			r.Logger.Errorf(err.Error())
		}
	}

	// Trigger Lifecycle Callbacks
	for _, deployer := range deployers {
		if err := deployer.TriggerLifecycleCallbacks(r.Builder.Config); err != nil {
			r.Logger.Errorf(err.Error())
		}
	}

	// Clear previous Version
	for _, deployer := range deployers {
		if err := deployer.CleanPreviousVersion(r.Builder.Config); err != nil {
			r.Logger.Errorf(err.Error())
		}
	}

	// Checking all previous version before delete asg
	cleanChecking(deployers, r.Builder.Config)

	// gather metrics of previous version
	for _, deployer := range deployers {
		if err := deployer.GatherMetrics(r.Builder.Config); err != nil {
			r.Logger.Errorf(err.Error())
		}
	}

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
func doHealthchecking(deployers []deployer.DeployManager, config builder.Config, logger *Logger.Logger) error {
	healthyStackList := []string{}
	healthy := false

	ch := make(chan map[string]bool)

	for !healthy {
		count := 0

		logger.Debugf("Start Timestamp: %d, timeout: %s", config.StartTimestamp, config.Timeout)
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("Timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

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
			time.Sleep(config.PollingInterval)
		}
	}

	return nil
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
			time.Sleep(config.PollingInterval)
		}
	}
}

func filterS3Path(path string) (string, string) {
	path = strings.ReplaceAll(path, builder.S3_PREFIX, "")
	split := strings.Split(path, "/")

	return split[0], strings.Join(split[1:], "/")
}
