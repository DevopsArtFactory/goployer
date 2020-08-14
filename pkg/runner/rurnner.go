package runner

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/slack"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/collector"
	"github.com/DevopsArtFactory/goployer/pkg/deployer"
	"github.com/DevopsArtFactory/goployer/pkg/initializer"
	"github.com/DevopsArtFactory/goployer/pkg/inspector"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Runner struct {
	Logger     *Logger.Logger
	Builder    builder.Builder
	Collector  collector.Collector
	Slacker    slack.Slack
	FuncMapper map[string]func() error
}

func SetupBuilder(mode string) (builder.Builder, error) {
	// Create new builder
	builderSt, err := builder.NewBuilder(nil)
	if err != nil {
		return builder.Builder{}, err
	}

	if !checkMode(mode) {
		return builderSt, nil
	}

	if err := builderSt.PreConfigValidation(); err != nil {
		return builderSt, err
	}

	builderSt, err = setManifestToBuilder(builderSt)
	if err != nil {
		return builder.Builder{}, err
	}

	m, err := builder.ParseMetricConfig(builderSt.Config.DisableMetrics, builder.METRIC_YAML_PATH)
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

	m, err := builder.ParseMetricConfig(builderSt.Config.DisableMetrics, builder.METRIC_YAML_PATH)
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
		fileBytes, err := s.S3Service.GetManifest(FilterS3Path(builderSt.Config.Manifest))
		if err != nil {
			return builder.Builder{}, err
		}
		builderSt = builderSt.SetManifestConfigWithS3(fileBytes)
	}

	return builderSt, nil
}

// Initialize creates necessary files for goployer
func Initialize(args []string) error {
	var appName string
	var err error

	// validation
	if len(args) > 1 {
		return fmt.Errorf("usage: goployer init <application name>")
	}

	if len(args) == 0 {
		appName, err = getApplicationName()
		if err != nil {
			return err
		}
	} else {
		appName = args[0]
	}

	i := initializer.NewInitializer(appName)
	i.Logger.SetLevel(tool.LogLevelMapper[viper.GetString("log-level")])

	if err := i.RunInit(); err != nil {
		return err
	}

	return nil
}

// AddManifest creates single manifest file
func AddManifest(args []string) error {
	var appName string
	var err error

	// validation
	if len(args) > 1 {
		return fmt.Errorf("usage: goployer add <application name>")
	}

	if len(args) == 0 {
		appName, err = getApplicationName()
		if err != nil {
			return err
		}
	} else {
		appName = args[0]
	}

	i := initializer.NewInitializer(appName)
	i.Logger.SetLevel(tool.LogLevelMapper[viper.GetString("log-level")])

	if err := i.RunAdd(); err != nil {
		return err
	}

	return nil
}

//Start function is the starting point of all processes.
func Start(builderSt builder.Builder, mode string) error {

	if checkMode(mode) {
		// Check validation of configurations
		if err := builderSt.CheckValidation(); err != nil {
			return err
		}
	}

	// run with runner
	return withRunner(builderSt, mode, func(slacker slack.Slack) error {
		if !builderSt.Config.SlackOff {
			// These are post actions after deployment
			if mode == "deploy" {
				slacker.SendSimpleMessage(":100: Deployment is done.", builderSt.Config.Env)
			}

			if mode == "delete" {
				slacker.SendSimpleMessage(":100: Delete process is done.", "")
			}
		}

		return nil
	})
}

//withRunner creates runner and runs the deployment process
func withRunner(builderSt builder.Builder, mode string, postAction func(slacker slack.Slack) error) error {
	runner, err := NewRunner(builderSt, mode)
	if err != nil {
		return err
	}
	runner.LogFormatting(builderSt.Config.LogLevel)

	if err := runner.Run(mode); err != nil {
		return err
	}

	return postAction(runner.Slacker)
}

//NewRunner creates a new runner
func NewRunner(newBuilder builder.Builder, mode string) (Runner, error) {
	newRunner := Runner{
		Logger:  Logger.New(),
		Builder: newBuilder,
		Slacker: slack.NewSlackClient(newBuilder.Config.SlackOff),
	}

	if checkMode(mode) {
		newRunner.Collector = collector.NewCollector(newBuilder.MetricConfig, newBuilder.Config.AssumeRole)
	}

	newRunner.FuncMapper = map[string]func() error{
		"deploy": newRunner.Deploy,
		"delete": newRunner.Delete,
		"status": newRunner.Status,
	}

	return newRunner, nil
}

// Set log format
func (r Runner) LogFormatting(logLevel string) {
	r.Logger.SetOutput(os.Stdout)
	r.Logger.SetLevel(tool.LogLevelMapper[logLevel])
}

// Run executes all required steps for deployments
func (r Runner) Run(mode string) error {
	f, ok := r.FuncMapper[mode]
	if !ok {
		return fmt.Errorf("no function exists to run for %s", mode)
	}
	return f()
}

func (r Runner) Deploy() error {
	defer func() {
		if err := recover(); err != nil {
			Logger.Error(err)
			os.Exit(1)
		}
	}()

	if err := r.LocalCheck("Do you really want to deploy this application? "); err != nil {
		return err
	}

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
		if err := r.CheckEnabledMetrics(); err != nil {
			return err
		}
	}

	r.Logger.Debugf("create wait group for deployer setup")
	wg := sync.WaitGroup{}

	//Prepare deployers
	r.Logger.Debug("create deployers for stacks")
	deployers := []deployer.DeployManager{}
	for _, stack := range r.Builder.Stacks {
		if r.Builder.Config.Stack != "" && stack.Stack != r.Builder.Config.Stack {
			r.Logger.Debugf("Skipping this stack, stack=%s", stack.Stack)
			continue
		}

		r.Logger.Debugf("add deployer setup function : %s", stack.Stack)
		deployers = append(deployers, getDeployer(r.Logger, stack, r.Builder.AwsConfig, r.Slacker, r.Collector))
	}

	r.Logger.Debugf("successfully assign deployer to stacks")

	// Check Previous Version
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.CheckPrevious(r.Builder.Config); err != nil {
				r.Logger.Errorf("[STEP_CHECK_PREVIOUS] check previous deployer error occurred: %s", err.Error())
			}

			if err := deployer.Deploy(r.Builder.Config); err != nil {
				r.Logger.Errorf("[STEP_DEPLOY] deploy step error occurred: %s", err.Error())
			}
		}(d)
	}

	wg.Wait()

	// healthcheck
	if err := doHealthchecking(deployers, r.Builder.Config, r.Logger); err != nil {
		return err
	}

	// Attach scaling policy
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.FinishAdditionalWork(r.Builder.Config); err != nil {
				r.Logger.Errorf(err.Error())
			}

			if err := deployer.TriggerLifecycleCallbacks(r.Builder.Config); err != nil {
				r.Logger.Errorf(err.Error())
			}

			if err := deployer.CleanPreviousVersion(r.Builder.Config); err != nil {
				r.Logger.Errorf(err.Error())
			}
		}(d)
	}

	wg.Wait()

	// Checking all previous version before delete asg
	cleanChecking(deployers, r.Builder.Config)

	// gather metrics of previous version
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.GatherMetrics(r.Builder.Config); err != nil {
				r.Logger.Errorf(err.Error())
			}
		}(d)
	}
	wg.Wait()

	return nil
}

func (r Runner) Delete() error {
	defer func() {
		if err := recover(); err != nil {
			Logger.Error(err)
			os.Exit(1)
		}
	}()

	if err := r.LocalCheck("Do you really want to delete applications? "); err != nil {
		return err
	}

	//Send Beginning Message
	r.Logger.Info("Beginning delete process: ", r.Builder.AwsConfig.Name)
	r.Builder.Config.SlackOff = true

	if r.Builder.MetricConfig.Enabled {
		if err := r.CheckEnabledMetrics(); err != nil {
			return err
		}
	}

	wg := sync.WaitGroup{}

	//Prepare deployers
	r.Logger.Debug("create deployers for stacks to delete")
	deployers := []deployer.DeployManager{}
	for _, stack := range r.Builder.Stacks {
		// If target stack is passed from command, then
		// Skip other stacks
		if r.Builder.Config.Stack != "" && stack.Stack != r.Builder.Config.Stack {
			r.Logger.Debugf("Skipping this stack, stack=%s", stack.Stack)
			continue
		}

		r.Logger.Debugf("add deployer setup function : %s", stack.Stack)
		d := getDeployer(r.Logger, stack, r.Builder.AwsConfig, r.Slacker, r.Collector)
		deployers = append(deployers, d)
	}

	r.Logger.Debugf("successfully assign deployer to stacks")

	// Check Previous Version
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.CheckPrevious(r.Builder.Config); err != nil {
				r.Logger.Errorf("[STEP_CHECK_PREVIOUS] check previous deployer error occurred: %s", err.Error())
			}

			deployer.SkipDeployStep()

			// Trigger Lifecycle Callbacks
			if err := deployer.TriggerLifecycleCallbacks(r.Builder.Config); err != nil {
				r.Logger.Errorf(err.Error())
			}

			// Clear previous Version
			if err := deployer.CleanPreviousVersion(r.Builder.Config); err != nil {
				r.Logger.Errorf(err.Error())
			}

		}(d)
	}
	wg.Wait()

	// Checking all previous version before delete asg
	cleanChecking(deployers, r.Builder.Config)

	// gather metrics of previous version
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.GatherMetrics(r.Builder.Config); err != nil {
				r.Logger.Errorf(err.Error())
			}
		}(d)
	}
	wg.Wait()

	return nil
}

func (r Runner) Status() error {
	inspector := inspector.New(r.Builder.Config.Region)

	asg, err := inspector.SelectStack(r.Builder.Config.Application)
	if err != nil {
		return err
	}

	group, err := inspector.GetStackInformation(asg)
	if err != nil {
		return err
	}

	inspector.StatusSummary = inspector.SetStatusSummary(group)

	if err := inspector.Print(); err != nil {
		return err
	}

	return nil
}

//Generate new deployer
func getDeployer(logger *Logger.Logger, stack schemas.Stack, awsConfig schemas.AWSConfig, slack slack.Slack, c collector.Collector) deployer.DeployManager {
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

		for _, d := range deployers {
			if tool.IsStringInArray(d.GetStackName(), healthyStackList) {
				continue
			}

			count += 1

			//Start healthcheck thread
			go func(deployer deployer.DeployManager) {
				ch <- deployer.HealthChecking(config)
			}(d)
		}

		for count > 0 {
			ret := <-ch
			if ret["error"] {
				return fmt.Errorf("error happened while healthchecking")
			}
			for key, val := range ret {
				if key == "error" {
					continue
				}
				if val {
					healthyStackList = append(healthyStackList, key)
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
		for _, d := range deployers {
			if tool.IsStringInArray(d.GetStackName(), doneStackList) {
				continue
			}

			count += 1

			//Start terminateChecking thread
			go func(deployer deployer.DeployManager) {
				ch <- deployer.TerminateChecking(config)
			}(d)
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

func (r Runner) CheckEnabledMetrics() error {
	r.Logger.Infof("Metric Measurement is enabled")

	r.Logger.Debugf("check if storage exists or not")
	if err := r.Collector.CheckStorage(r.Logger); err != nil {
		return err
	}

	return nil
}

func FilterS3Path(path string) (string, string) {
	path = strings.ReplaceAll(path, builder.S3_PREFIX, "")
	split := strings.Split(path, "/")

	return split[0], strings.Join(split[1:], "/")
}

func getApplicationName() (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "What is application name? ",
	}
	survey.AskOne(prompt, &answer)
	if answer == "" {
		return "", fmt.Errorf("canceled")
	}

	return answer, nil
}

func checkMode(mode string) bool {
	return tool.IsStringInArray(mode, []string{"deploy", "delete"})
}

func (r Runner) LocalCheck(message string) error {
	// From local os, you need to ensure that this command is intended
	if runtime.GOOS == "darwin" && !r.Builder.Config.AutoApply {
		if !tool.AskContinue(message) {
			return fmt.Errorf("you declined to run command")
		}
	}
	return nil
}
