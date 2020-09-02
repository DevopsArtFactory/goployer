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
	"context"
	"io"

	"github.com/spf13/cobra"
)

type Command interface {
	WithDescription(description string) Command
	WithLongDescription(description string) Command
	SetFlags() Command
	SetPreRunWithArgs(action func(context.Context, io.Writer, []string) error) Command
	RunWithNoArgs(action func(context.Context, io.Writer, string) error) *cobra.Command
	RunWithArgs(action func(context.Context, io.Writer, []string, string) error) *cobra.Command
}

type command struct {
	cmd cobra.Command
}

// NewCmd creates a new command builder.
func NewCmd(use string) Command {
	return &command{
		cmd: cobra.Command{
			Use: use,
		},
	}
}

// Write short description
func (c command) WithDescription(description string) Command {
	c.cmd.Short = description
	return c
}

// Write long description
func (c command) WithLongDescription(description string) Command {
	c.cmd.Long = description
	return c
}

// SetFlags set flags for commands
func (c command) SetFlags() Command {
	SetCommandFlags(&c.cmd)
	return c
}

// Set prerun with argument
func (c command) SetPreRunWithArgs(function func(context.Context, io.Writer, []string) error) Command {
	c.cmd.PreRunE = func(_ *cobra.Command, args []string) error {
		return funcError(function(c.cmd.Context(), c.cmd.OutOrStderr(), args))
	}
	return c
}

//Run command without Argument
func (c command) RunWithNoArgs(function func(context.Context, io.Writer, string) error) *cobra.Command {
	c.cmd.Args = cobra.NoArgs
	c.cmd.RunE = func(*cobra.Command, []string) error {
		return funcError(function(c.cmd.Context(), c.cmd.OutOrStderr(), c.cmd.Use))
	}
	return &c.cmd
}

// Run command with extra arguments
func (c command) RunWithArgs(function func(context.Context, io.Writer, []string, string) error) *cobra.Command {
	c.cmd.RunE = func(_ *cobra.Command, args []string) error {
		return funcError(function(c.cmd.Context(), c.cmd.OutOrStderr(), args, c.cmd.Use))
	}
	return &c.cmd
}

// Handle Error from real function
func funcError(err error) error {
	return err
}
