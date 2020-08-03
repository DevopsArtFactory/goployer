package cmd

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"reflect"
	"time"
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
	noString        = ""
	timeout         = 60 * time.Minute
	pollingInterval = 60 * time.Second
)

var zeroTimeout = 0 * time.Minute
var zeroPollingInterval = 0 * time.Second

var flagKey = map[string]string{
	"deploy": "fullset",
	"delete": "fullset",
	"init":   "initset",
}

var FlagRegistry = map[string][]Flag{
	"fullset": []Flag{
		{
			Name:          "manifest",
			Shorthand:     "m",
			Usage:         "The manifest configuration file to use. (required)",
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "manifest-s3-region",
			Usage:         "Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)",
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "stack",
			Usage:         "stack that should be deployed.(required)",
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "ami",
			Usage:         "Amazon AMI to use.",
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "env",
			Usage:         "The environment that is being deployed into.",
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "assume-role",
			Usage:         "The Role ARN to assume into.",
			Value:         aws.String(noString),
			DefValue:      "",
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
			Value:         aws.String(noString),
			DefValue:      "",
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
			Value:         aws.String(noString),
			DefValue:      "info",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "extra-tags",
			Usage:         "Extra tags to add to autoscaling group tags",
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "ansible-extra-vars",
			Usage:         "Extra variables for ansible",
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "override-instance-type",
			Usage:         "Instance Type to override",
			Value:         aws.String(noString),
			DefValue:      "",
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
			Value:         aws.String(noString),
			DefValue:      "",
			FlagAddMethod: "StringVar",
		},
		{
			Name:          "release-notes-base64",
			Usage:         "Base64 encoded string of release note for the current deployment",
			Value:         aws.String(noString),
			DefValue:      "",
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
	},
	"initset": []Flag{
		{
			Name:          "log-level",
			Usage:         "Level of logging",
			Value:         aws.String(noString),
			DefValue:      "info",
			FlagAddMethod: "StringVar",
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
	if fl.Shorthand != "" {
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
	for i := range FlagRegistry[flagKey[cmd.Use]] {
		fl := &FlagRegistry[flagKey[cmd.Use]][i]
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
