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
	"fmt"
	"io"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/version"
)

var (
	cfgFile string
	v       string
)

// NewRootCommand creates new root command
func NewRootCommand(_, stderr io.Writer) *cobra.Command {
	cobra.OnInitialize(initConfig)
	rootCmd := &cobra.Command{
		Use:   "goployer",
		Short: "A brief description of your application",
		Long: `Goployer is a command line tool to deploy applications on AWS.

You can find more information in https://goployer.dev`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Setup logs
			if err := setUpLogs(stderr, v); err != nil {
				return err
			}

			version := version.Get()

			logrus.Infof("Goployer %+v", version)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(NewDeployCommand())
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewDeleteCommand())
	rootCmd.AddCommand(NewInitCommand())
	rootCmd.AddCommand(NewStatusCommand())
	rootCmd.AddCommand(NewAddCommand())
	rootCmd.AddCommand(NewUpdateCommand())

	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", constants.DefaultLogLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")

	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
	}

	viper.AutomaticEnv() // read in environment variables that match
}

// setUpLogs setup log level
func setUpLogs(stdErr io.Writer, level string) error {
	logrus.SetOutput(stdErr)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}
	logrus.SetLevel(lvl)
	return nil
}
