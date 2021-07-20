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

package cmd

import (
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
)

type Flag struct {
	Name               string
	Shorthand          string
	Usage              string
	Value              interface{}
	DefValue           interface{}
	DefValuePerCommand map[string]interface{}
	FlagAddMethod      string
	Hidden             bool

	pflag *pflag.Flag
}

const (
	timeout         = 60 * time.Minute
	pollingInterval = 60 * time.Second
)

var zeroTimeout = 0 * time.Minute
var zeroPollingInterval = 0 * time.Second

var flagKey = map[string]string{
	"deploy":  "deploySet",
	"delete":  "fullSet",
	"init":    "initSet",
	"status":  "statusSet",
	"update":  "updateSet",
	"add":     "addSet",
	"refresh": "refreshSet",
}

var CommonFlagRegistry = []Flag{
	{
		Name:          "profile",
		Shorthand:     "p",
		Usage:         "Profile configuration of AWS",
		Value:         aws.String(constants.EmptyString),
		DefValue:      constants.EmptyString,
		FlagAddMethod: "StringVar",
	},
}

var FlagRegistry = map[string][]Flag{
	"fullSet": {
		{
			Name:          "manifest",
			Shorthand:     "m",
			Usage:         "The manifest configuration file to use. (required)",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "manifest-s3-region",
			Usage:         "Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "stack",
			Usage:         "stack that should be deployed.(required)",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "ami",
			Usage:         "Amazon AMI to use.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "env",
			Usage:         "The environment that is being deployed into.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "assume-role",
			Usage:         "The Role ARN to assume into.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "timeout",
			Usage:         "Time to wait for deploy to finish before timing out (default 60m)",
			Value:         &zeroTimeout,
			DefValue:      timeout,
			FlagAddMethod: "DurationVar",
		},
		{
			Name:          "region",
			Usage:         "The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "slack-off",
			Usage:         "Turn off slack alarm",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "log-level",
			Usage:         "Level of logging",
			Shorthand:     "v",
			Value:         aws.String(constants.EmptyString),
			DefValue:      "warning",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "extra-tags",
			Usage:         "Extra tags to add to autoscaling group tags",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "ansible-extra-vars",
			Usage:         "Extra variables for ansible",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "override-instance-type",
			Usage:         "Instance Type to override",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "override-spot-types",
			Usage:         "Spot Instance Type to override",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "disable-metrics",
			Usage:         "Disable gathering metrics.",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "release-notes",
			Usage:         "Release note for the current deployment",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "release-notes-base64",
			Usage:         "Base64 encoded string of release note for the current deployment",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "force-manifest-capacity",
			Usage:         "Force-apply the capacity of instances in the manifest file",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "polling-interval",
			Usage:         "Time to interval for polling health check (default 60s)",
			Value:         &zeroPollingInterval,
			DefValue:      pollingInterval,
			FlagAddMethod: "DurationVar",
		},
		{
			Name:          "auto-apply",
			Usage:         "Apply command without confirmation from local terminal",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
	},
	"deploySet": {
		{
			Name:          "manifest",
			Shorthand:     "m",
			Usage:         "The manifest configuration file to use. (required)",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "manifest-s3-region",
			Usage:         "Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "stack",
			Usage:         "stack that should be deployed.(required)",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "ami",
			Usage:         "Amazon AMI to use.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "env",
			Usage:         "The environment that is being deployed into.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "assume-role",
			Usage:         "The Role ARN to assume into.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "timeout",
			Usage:         "Time to wait for deploy to finish before timing out (default 60m)",
			Value:         &zeroTimeout,
			DefValue:      timeout,
			FlagAddMethod: "DurationVar",
		},
		{
			Name:          "region",
			Usage:         "The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "slack-off",
			Usage:         "Turn off slack alarm",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "log-level",
			Usage:         "Level of logging",
			Shorthand:     "v",
			Value:         aws.String(constants.EmptyString),
			DefValue:      "warning",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "extra-tags",
			Usage:         "Extra tags to add to autoscaling group tags",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "ansible-extra-vars",
			Usage:         "Extra variables for ansible",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "override-instance-type",
			Usage:         "Instance Type to override",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "override-spot-types",
			Usage:         "Spot Instance Type to override",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "disable-metrics",
			Usage:         "Disable gathering metrics.",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "release-notes",
			Usage:         "Release note for the current deployment",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "release-notes-base64",
			Usage:         "Base64 encoded string of release note for the current deployment",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "force-manifest-capacity",
			Usage:         "Force-apply the capacity of instances in the manifest file",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "polling-interval",
			Usage:         "Time to interval for polling health check (default 60s)",
			Value:         &zeroPollingInterval,
			DefValue:      pollingInterval,
			FlagAddMethod: "DurationVar",
		},
		{
			Name:          "auto-apply",
			Usage:         "Apply command without confirmation from local terminal",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "complete-canary",
			Usage:         "Complete the rest of canary deployment.(Only works with Canary replacement type)",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
	},
	"initSet": {
		{
			Name:          "log-level",
			Usage:         "Level of logging",
			Shorthand:     "v",
			Value:         aws.String(constants.EmptyString),
			DefValue:      "warning",
			FlagAddMethod: "StringVar",
		},
	},
	"addSet": {
		{
			Name:          "log-level",
			Usage:         "Level of logging",
			Shorthand:     "v",
			Value:         aws.String(constants.EmptyString),
			DefValue:      "warning",
			FlagAddMethod: "StringVar",
		},
	},
	"statusSet": {
		{
			Name:          "region",
			Usage:         "Region of autoscaling group",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
	},
	"updateSet": {
		{
			Name:          "region",
			Usage:         "Region of autoscaling group",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "auto-apply",
			Usage:         "Apply command without confirmation from local terminal",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "min",
			Usage:         "Minimum instance capacity you want to update with",
			Value:         aws.Int(-1),
			DefValue:      -1,
			FlagAddMethod: "IntVar",
		},
		{
			Name:          "max",
			Usage:         "Maximum instance capacity you want to update with",
			Value:         aws.Int(-1),
			DefValue:      -1,
			FlagAddMethod: "IntVar",
		},
		{
			Name:          "desired",
			Usage:         "Desired instance capacity you want to update with",
			Value:         aws.Int(-1),
			DefValue:      -1,
			FlagAddMethod: "IntVar",
		},
		{
			Name:          "polling-interval",
			Usage:         "Time to interval for polling health check (default 60s)",
			Value:         &zeroPollingInterval,
			DefValue:      pollingInterval,
			FlagAddMethod: "DurationVar",
		},
		{
			Name:          "timeout",
			Usage:         "Time to wait for deploy to finish before timing out (default 60m)",
			Value:         &zeroTimeout,
			DefValue:      timeout,
			FlagAddMethod: "DurationVar",
		},
	},
	"refreshSet": {
		{
			Name:          "region",
			Usage:         "Region of autoscaling group",
			Value:         aws.String(constants.EmptyString),
			DefValue:      constants.EmptyString,
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "auto-apply",
			Usage:         "Apply command without confirmation from local terminal",
			Value:         aws.Bool(false),
			DefValue:      false,
			FlagAddMethod: "BoolVar",
		},
		{
			Name:          "instance-warmup",
			Usage:         "How much time it takes a newly launched instance to be ready to use.",
			Value:         aws.Int(constants.DefaultInstanceWarmup),
			DefValue:      constants.DefaultInstanceWarmup,
			FlagAddMethod: "IntVar",
		},
		{
			Name:          "min-healthy-percentage",
			Usage:         "At least this percentage of the desired capacity of the Auto Scaling group must remain healthy during this operation to allow it to continue.",
			Value:         aws.Int(constants.DefaultMinHealthyPercentage),
			DefValue:      constants.DefaultMinHealthyPercentage,
			FlagAddMethod: "IntVar",
		},
		{
			Name:          "polling-interval",
			Usage:         "Time to interval for polling health check (default 60s)",
			Value:         &zeroPollingInterval,
			DefValue:      pollingInterval,
			FlagAddMethod: "DurationVar",
		},
		{
			Name:          "timeout",
			Usage:         "Time to wait for deploy to finish before timing out (default 60m)",
			Value:         &zeroTimeout,
			DefValue:      timeout,
			FlagAddMethod: "DurationVar",
		},
	},
}

func (fl *Flag) flag() *pflag.Flag {
	if fl.pflag != nil {
		return fl.pflag
	}

	inputs := []interface{}{fl.Value, fl.Name}
	if fl.FlagAddMethod != "Var" {
		inputs = append(inputs, fl.DefValue)
	}
	inputs = append(inputs, fl.Usage)

	fs := pflag.NewFlagSet(fl.Name, pflag.ContinueOnError)
	reflect.ValueOf(fs).MethodByName(fl.FlagAddMethod).Call(reflectValueOf(inputs))
	f := fs.Lookup(fl.Name)
	if fl.Shorthand != constants.EmptyString {
		f.Shorthand = fl.Shorthand
	}
	f.Hidden = fl.Hidden
	fl.pflag = f

	return f
}

func reflectValueOf(values []interface{}) []reflect.Value {
	var results []reflect.Value
	for _, v := range values {
		results = append(results, reflect.ValueOf(v))
	}
	return results
}

//Add command flags
func SetCommandFlags(cmd *cobra.Command) {
	var flagsForCommand []*Flag

	FlagRegistries := append(CommonFlagRegistry, FlagRegistry[flagKey[cmd.Use]]...)
	for i := range FlagRegistries {
		fl := &FlagRegistries[i]
		cmd.PersistentFlags().AddFlag(fl.flag())
		flagsForCommand = append(flagsForCommand, fl)
	}

	// Apply command-specific default values to flags.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Update default values.
		for _, fl := range flagsForCommand {
			viper.BindPFlag(fl.Name, cmd.PersistentFlags().Lookup(fl.Name))
		}

		if parent := cmd.Parent(); parent != nil {
			if preRun := parent.PersistentPreRunE; preRun != nil {
				if err := preRun(cmd, args); err != nil {
					return err
				}
			} else if preRun := parent.PersistentPreRun; preRun != nil {
				preRun(cmd, args)
			}
		}

		return nil
	}
}
