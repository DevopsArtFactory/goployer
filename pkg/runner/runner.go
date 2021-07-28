/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package runner

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/GwonsooLee/kubenx/pkg/color"
	Logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/collector"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/deployer"
	"github.com/DevopsArtFactory/goployer/pkg/helper"
	"github.com/DevopsArtFactory/goployer/pkg/initializer"
	"github.com/DevopsArtFactory/goployer/pkg/inspector"
	"github.com/DevopsArtFactory/goployer/pkg/refresh"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/slack"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type Runner struct {
	Logger     *Logger.Logger
	Builder    builder.Builder
	Collector  collector.Collector
	Slacker    slack.Slack
	FuncMapper map[string]func() error
}

// NewRunner creates a new runner
func NewRunner(newBuilder builder.Builder, mode string) (Runner, error) {
	newRunner := Runner{
		Logger:  Logger.New(),
		Builder: newBuilder,
		Slacker: slack.NewSlackClient(newBuilder.Config.SlackOff),
	}

	if checkBuilderConfigurationNeeded(mode) {
		newRunner.Collector = collector.NewCollector(newBuilder.MetricConfig, newBuilder.Config.AssumeRole)
	}

	newRunner.FuncMapper = map[string]func() error{
		"deploy":  newRunner.Deploy,
		"delete":  newRunner.Delete,
		"status":  newRunner.Status,
		"update":  newRunner.Update,
		"refresh": newRunner.Refresh,
	}

	return newRunner, nil
}

// SetupBuilder setup builder struct for configuration
func SetupBuilder(mode string) (builder.Builder, error) {
	// Create new builder
	builderSt, err := builder.NewBuilder(nil)
	if err != nil {
		return builder.Builder{}, err
	}

	if !checkBuilderConfigurationNeeded(mode) {
		return builderSt, nil
	}

	if err := builderSt.PreConfigValidation(); err != nil {
		return builderSt, err
	}

	builderSt, err = setManifestToBuilder(builderSt)
	if err != nil {
		return builder.Builder{}, err
	}

	m, err := builder.ParseMetricConfig(builderSt.Config.DisableMetrics, constants.MetricYamlPath)
	if err != nil {
		return builder.Builder{}, err
	}

	builderSt.MetricConfig = m

	return builderSt, nil
}

// ServerSetup setup a goployer server
func ServerSetup(config schemas.Config) (builder.Builder, error) {
	// Create new builder
	builderSt, err := builder.NewBuilder(&config)
	if err != nil {
		return builder.Builder{}, err
	}

	builderSt, err = setManifestToBuilder(builderSt)
	if err != nil {
		return builder.Builder{}, err
	}

	m, err := builder.ParseMetricConfig(builderSt.Config.DisableMetrics, constants.MetricYamlPath)
	if err != nil {
		return builder.Builder{}, err
	}

	builderSt.MetricConfig = m

	return builderSt, nil
}

// setManifestToBuilder creates builderSt with manifest configurations
func setManifestToBuilder(builderSt builder.Builder) (builder.Builder, error) {
	if !strings.HasPrefix(builderSt.Config.Manifest, constants.S3Prefix) {
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
		return errors.New("usage: goployer init <application name>")
	}

	if len(args) == 0 {
		appName, err = askApplicationName()
		if err != nil {
			return err
		}
	} else {
		appName = args[0]
	}

	i := initializer.NewInitializer(appName)
	i.Logger.SetLevel(constants.LogLevelMapper[viper.GetString("log-level")])

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
		return errors.New("usage: goployer add <application name>")
	}

	if len(args) == 0 {
		appName, err = askApplicationName()
		if err != nil {
			return err
		}
	} else {
		appName = args[0]
	}

	i := initializer.NewInitializer(appName)
	i.Logger.SetLevel(constants.LogLevelMapper[viper.GetString("log-level")])

	if err := i.RunAdd(); err != nil {
		return err
	}

	return nil
}

// Start function is the starting point of all processes.
func Start(builderSt builder.Builder, mode string) error {
	if checkBuilderConfigurationNeeded(mode) {
		// Check validation of configurations
		if err := builderSt.CheckValidation(); err != nil {
			return err
		}
	}

	// run with runner
	return withRunner(builderSt, mode, func(slacker slack.Slack) error {
		// These are post actions after deployment
		if !builderSt.Config.SlackOff {
			if mode == "deploy" {
				slacker.SendSimpleMessage(fmt.Sprintf(":100: Deployment is done: %s", builderSt.AwsConfig.Name))
			}

			if mode == "delete" {
				slacker.SendSimpleMessage(fmt.Sprintf(":100: Delete process is done: %s", builderSt.AwsConfig.Name))
			}
		}

		return nil
	})
}

// withRunner creates runner and runs the deployment process
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

// LogFormatting sets log format
func (r Runner) LogFormatting(logLevel string) {
	r.Logger.SetOutput(os.Stdout)
	r.Logger.SetLevel(constants.LogLevelMapper[logLevel])
}

// Run executes all required steps for deployments
func (r Runner) Run(mode string) error {
	f, ok := r.FuncMapper[mode]
	if !ok {
		return fmt.Errorf("no function exists to run for %s", mode)
	}
	return f()
}

// Deploy is the main function of `goployer deploy`
func (r Runner) Deploy() error {
	out := os.Stdout
	defer func() {
		if err := recover(); err != nil {
			Logger.Error(err)
			os.Exit(1)
		}
	}()

	if err := tool.LocalCheck("Do you really want to deploy this application? ", r.Builder.Config.AutoApply); err != nil {
		return err
	}

	//Send Beginning Message
	r.Logger.Infof("Beginning deployment: %s", r.Builder.AwsConfig.Name)

	if err := r.Builder.PrintSummary(out, r.Builder.Config.Stack, r.Builder.Config.Region); err != nil {
		return err
	}

	if r.Slacker.ValidClient() {
		r.Logger.Debug("Slack configuration is valid")
		var stacks []schemas.Stack
		for _, s := range r.Builder.Stacks {
			if len(r.Builder.Config.Stack) == 0 || r.Builder.Config.Stack == s.Stack {
				stacks = append(stacks, s)
			}
		}
		if err := r.Slacker.SendSummaryMessage(r.Builder.Config, stacks, r.Builder.AwsConfig.Name); err != nil {
			r.Logger.Warn(err.Error())
			r.Slacker.SlackOff = true
		}
	} else if !r.Builder.Config.SlackOff {
		// Slack variables are not set
		r.Logger.Warn("no slack variables exists. [ SLACK_TOKEN, SLACK_CHANNEL or SLACK_WEBHOOK_URL ]")
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
	var deployers []deployer.DeployManager
	for _, stack := range r.Builder.Stacks {
		if r.Builder.Config.Stack != "" && stack.Stack != r.Builder.Config.Stack {
			r.Logger.Debugf("Skipping this stack, stack=%s", stack.Stack)
			continue
		}

		r.Logger.Debugf("add deployer setup function : %s", stack.Stack)
		deployers = append(deployers, getDeployer(r.Logger, stack, r.Builder.AwsConfig, r.Builder.APITestTemplates, r.Builder.Config.Region, r.Slacker, r.Collector))
	}
	r.Logger.Debugf("successfully assign deployer to stacks")

	errs := make(chan error)
	// Check Previous Version
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.CheckPreviousResources(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepCheckPrevious] check previous deployer error occurred: %s", err.Error())
				errs <- err
			}

			if err := deployer.Deploy(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepDeploy] deploy step error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag := checkError(errs)
	if errFlag != nil {
		return errFlag
	}
	// Health checking step
	errs = make(chan error)
	fmt.Println("get int HealthChecking")
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.HealthChecking(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepHealthCheck] check previous deployer error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag = checkError(errs)
	if errFlag != nil {
		return errFlag
	}
	//AdditionalWork
	errs = make(chan error)
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			// Attach scaling policy
			if err := deployer.FinishAdditionalWork(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepFinishAdditionalWork] finish additional work error occurred: %s", err.Error())
				errs <- err
			}

			if err := deployer.TriggerLifecycleCallbacks(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepTriggerLifecycleCallbacks] trigger lifecycle callbacks error occurred: %s", err.Error())
				errs <- err
			}

			if err := deployer.CleanPreviousVersion(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepCleanPreviousVersion] clean previous verson error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag = checkError(errs)
	if errFlag != nil {
		return errFlag
	}
	//CleanChecking
	errs = make(chan error)
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.CleanChecking(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepCleanChecking] clean checking error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag = checkError(errs)
	if errFlag != nil {
		return errFlag
	}
	// gather metrics of previous version
	errs = make(chan error)
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.GatherMetrics(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepGatherMetrics] gather metrics error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag = checkError(errs)
	if errFlag != nil {
		return errFlag
	}
	// API Test
	errs = make(chan error)
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.RunAPITest(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepRunAPITest] API test error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag = checkError(errs)
	if errFlag != nil {
		return errFlag
	}
	return nil
}

// Delete is the main function for `goployer delete`
func (r Runner) Delete() error {
	defer func() {
		if err := recover(); err != nil {
			Logger.Error(err)
			os.Exit(1)
		}
	}()

	if err := tool.LocalCheck("Do you really want to delete applications? ", r.Builder.Config.AutoApply); err != nil {
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
	var deployers []deployer.DeployManager
	for _, stack := range r.Builder.Stacks {
		// If target stack is passed from command, then
		// Skip other stacks
		if r.Builder.Config.Stack != "" && stack.Stack != r.Builder.Config.Stack {
			r.Logger.Debugf("Skipping this stack, stack=%s", stack.Stack)
			continue
		}

		r.Logger.Debugf("add deployer setup function : %s", stack.Stack)
		d := getDeployer(r.Logger, stack, r.Builder.AwsConfig, r.Builder.APITestTemplates, r.Builder.Config.Region, r.Slacker, r.Collector)
		deployers = append(deployers, d)
	}

	r.Logger.Debugf("successfully assign deployer to stacks")

	// Check Previous Version
	errs := make(chan error)
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.GetDeployer().CheckPrevious(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepCheckPrevious] check previous deployer error occurred: %s", err.Error())
				errs <- err
			}

			deployer.GetDeployer().SkipDeployStep()

			// Trigger Lifecycle Callbacks
			if err := deployer.TriggerLifecycleCallbacks(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepTriggerLifecycleCallbacks] trigger lifecycle callbacks error occurred: %s", err.Error())
				errs <- err
			}

			// Clear previous Version
			if err := deployer.CleanPreviousVersion(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepCleanPreviousVersion] clean previous version error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag := checkError(errs)
	if errFlag != nil {
		return errFlag
	}

	errs = make(chan error)
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.CleanChecking(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepCleanChecking] clean checking error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag = checkError(errs)
	if errFlag != nil {
		return errFlag
	}
	// gather metrics of previous version
	errs = make(chan error)
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.GatherMetrics(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepGatherMetrics] gather metrics error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag = checkError(errs)
	if errFlag != nil {
		return errFlag
	}

	return nil
}

// Status shows the detailed information about autoscaling deployment
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

	launchTemplateInfo, err := inspector.GetLaunchTemplateInformation(*group.LaunchTemplate.LaunchTemplateId)
	if err != nil {
		return err
	}

	securityGroups, err := inspector.GetSecurityGroupsInformation(launchTemplateInfo.LaunchTemplateData.SecurityGroupIds)
	if err != nil {
		return err
	}

	inspector.StatusSummary = inspector.SetStatusSummary(group, securityGroups)

	if err := inspector.Print(); err != nil {
		return err
	}

	return nil
}

// Update will changes configuration of current deployment on live
func (r Runner) Update() error {
	var wg sync.WaitGroup
	i := inspector.New(r.Builder.Config.Region)

	asg, err := i.SelectStack(r.Builder.Config.Application)
	if err != nil {
		return err
	}

	group, err := i.GetStackInformation(asg)
	if err != nil {
		return err
	}

	oldCapacity := makeCapacityStruct(*group.MinSize, *group.MaxSize, *group.DesiredCapacity)
	newCapacity := makeCapacityStruct(nullCheck(r.Builder.Config.Min, oldCapacity.Min), nullCheck(r.Builder.Config.Max, oldCapacity.Max), nullCheck(r.Builder.Config.Desired, oldCapacity.Desired))
	if err := CheckUpdateInformation(oldCapacity, newCapacity); err != nil {
		return err
	}
	color.Cyan.Fprintln(os.Stdout, "[ AS IS ]")
	color.Cyan.Fprintf(os.Stdout, "Min: %d, Desired: %d, Max: %d", oldCapacity.Min, oldCapacity.Desired, oldCapacity.Max)
	color.Green.Fprintln(os.Stdout, "[ TO BE ]")
	color.Green.Fprintf(os.Stdout, "Min: %d, Desired: %d, Max: %d", newCapacity.Min, newCapacity.Desired, newCapacity.Max)

	if err := tool.LocalCheck("Do you really want to update? ", r.Builder.Config.AutoApply); err != nil {
		return err
	}

	if oldCapacity.Desired > newCapacity.Desired {
		r.Logger.Debugf("downsizing operation is triggered")
	}

	i.UpdateFields = inspector.UpdateFields{
		AutoscalingName: *group.AutoScalingGroupName,
		Capacity:        newCapacity,
	}

	r.Logger.Debugf("start updating configuration")
	if err := i.Update(); err != nil {
		return err
	}
	r.Logger.Debugf("update configuration is triggered")

	stack := i.GenerateStack(r.Builder.Config.Region, group)
	r.Builder.Config.DownSizingUpdate = oldCapacity.Desired > newCapacity.Desired
	r.Builder.Config.TargetAutoscalingGroup = i.UpdateFields.AutoscalingName
	r.Builder.Config.ForceManifestCapacity = false

	r.Logger.Debugf("create deployer for update")
	deployers := []deployer.DeployManager{
		getDeployer(r.Logger, stack, r.Builder.AwsConfig, r.Builder.APITestTemplates, r.Builder.Config.Region, r.Slacker, r.Collector),
	}

	// Health checking step
	errs := make(chan error)
	r.Logger.Debugf("Start health checking")
	for _, d := range deployers {
		wg.Add(1)
		go func(deployer deployer.DeployManager) {
			defer wg.Done()
			if err := deployer.HealthChecking(r.Builder.Config); err != nil {
				r.Logger.Errorf("[StepHealthCheck] check previous deployer error occurred: %s", err.Error())
				errs <- err
			}
		}(d)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()
	errFlag := checkError(errs)
	if errFlag != nil {
		return errFlag
	}

	r.Logger.Debugf("Health check process is done")
	r.Logger.Infof("update operation is finished")
	return nil
}

// Refresh will refresh autoscaling group instances
func (r Runner) Refresh() error {
	i := inspector.New(r.Builder.Config.Region)

	asg, err := i.SelectStack(r.Builder.Config.Application)
	if err != nil {
		return err
	}
	r.Logger.Debugf("Selected target autoscaling group: %s", asg)

	group, err := i.GetStackInformation(asg)
	if err != nil {
		return err
	}

	r.Logger.Debugf("Autoscaling group found: %s", *group.AutoScalingGroupARN)

	if err := tool.LocalCheck("Do you really want to refresh? ", r.Builder.Config.AutoApply); err != nil {
		return err
	}

	r.Logger.Debug("Create a new refresher")
	refresher := refresh.New(r.Builder.Config.Region)
	refresher.SetTarget(group)

	input := make(chan error)

	go func(ch chan error) {
		ch <- refresher.StartRefresh(r.Builder.Config.InstanceWarmup, r.Builder.Config.MinHealthyPercentage)
	}(input)
	time.Sleep(3 * time.Second)

	if err := refresher.StatusCheck(r.Builder.Config.PollingInterval, r.Builder.Config.Timeout); err != nil {
		return err
	}

	if err := refresher.PrintResult(); err != nil {
		return err
	}

	if err := <-input; err != nil {
		r.Logger.Warn(err.Error())
	}

	r.Logger.Infof("Refresh operation is finished")
	return nil
}

// Generate new deployer
func getDeployer(logger *Logger.Logger, stack schemas.Stack, awsConfig schemas.AWSConfig, apiTestTemplates []*schemas.APITestTemplate, region string, slack slack.Slack, c collector.Collector) deployer.DeployManager {
	var att *schemas.APITestTemplate
	if stack.APITestEnabled {
		for _, at := range apiTestTemplates {
			if at.Name == stack.APITestTemplate {
				att = at
				break
			}
		}
	}

	h := helper.DeployerHelper{
		Logger:           logger,
		Stack:            stack,
		AwsConfig:        awsConfig,
		APITestTemplates: att,
		Region:           region,
		Slack:            slack,
		Collector:        c,
	}

	var d deployer.DeployManager
	switch h.Stack.ReplacementType {
	case constants.BlueGreenDeployment:
		d = deployer.NewBlueGreen(&h)
	case constants.CanaryDeployment:
		d = deployer.NewCanary(&h)
	case constants.RollingUpdateDeployment:
		d = deployer.NewRollingUpdate(&h)
	case constants.DeployOnly:
		d = deployer.NewDeployOnly(&h)
	}

	return d
}

// CheckEnabledMetrics checks if metrics configuration is enabled or not
func (r Runner) CheckEnabledMetrics() error {
	r.Logger.Debugf("Check if storage exists or not")
	if err := r.Collector.CheckStorage(r.Logger); err != nil {
		return err
	}

	return nil
}

// FilterS3Path detects s3 path
func FilterS3Path(path string) (string, string) {
	path = strings.ReplaceAll(path, constants.S3Prefix, "")
	split := strings.Split(path, "/")

	return split[0], strings.Join(split[1:], "/")
}

// askApplicationName gets application name from interactive terminal
func askApplicationName() (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "What is application name? ",
	}
	survey.AskOne(prompt, &answer)
	if answer == constants.EmptyString {
		return constants.EmptyString, errors.New("canceled")
	}

	return answer, nil
}

// checkBuilderConfigurationNeeded checks if mode needs configuration settings like builder, metrics etc
func checkBuilderConfigurationNeeded(mode string) bool {
	return tool.IsStringInArray(mode, []string{"deploy", "delete"})
}

// CheckUpdateInformation checks if updated information is valid or not
func CheckUpdateInformation(old, new schemas.Capacity) error {
	if new.Min > new.Max {
		return errors.New("minimum value cannot be larger than maximum value")
	}

	if new.Min > new.Desired {
		return errors.New("desired value cannot be smaller than maximum value")
	}

	if new.Desired > new.Max {
		return errors.New("desired value cannot be larger than max value")
	}

	if old == new {
		return errors.New("nothing is updated")
	}
	return nil
}

// makeCapacityStruct creates schemas.Capacity with values
func makeCapacityStruct(min, max, desired int64) schemas.Capacity {
	return schemas.Capacity{
		Min:     min,
		Max:     max,
		Desired: desired,
	}
}

// nullCheck will return original value if no input exists
func nullCheck(input, origin int64) int64 {
	if input < 0 {
		return origin
	}

	return input
}

func checkError(errs chan error) error {
	if errs != nil {
		for err := range errs {
			if err != nil {
				return err
			}
		}
	}
	return nil
}
