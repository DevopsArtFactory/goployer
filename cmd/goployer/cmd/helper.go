package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"io"
)

type Command interface {
	WithDescription(description string) Command
	WithLongDescription(description string) Command
	//SetAliases(alias []string) Command
	//AddCommand(cmd *cobra.Command) Command
	//AddGetGroups() Command
	//AddSearchGroups() Command
	//AddInspectGroups() Command
	//AddConfigGroups() Command
	SetFlags() Command
	SetPreRunWithArgs(action func(context.Context, io.Writer, []string) error) Command
	RunWithNoArgs(action func(context.Context, io.Writer) error) *cobra.Command
	RunWithArgs(action func(context.Context, io.Writer, []string) error) *cobra.Command
	//RunWithArgsAndCmd(action func(context.Context, io.Writer, *cobra.Command, []string) error) *cobra.Command
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
func (c command) RunWithNoArgs(function func(context.Context, io.Writer) error) *cobra.Command {
	c.cmd.Args = cobra.NoArgs
	c.cmd.RunE = func(*cobra.Command, []string) error {
		return funcError(function(c.cmd.Context(), c.cmd.OutOrStderr()))
	}
	return &c.cmd
}

// Run command with extra arguments
func (c command) RunWithArgs(function func(context.Context, io.Writer, []string) error) *cobra.Command {
	c.cmd.RunE = func(_ *cobra.Command, args []string) error {
		return funcError(function(c.cmd.Context(), c.cmd.OutOrStderr(), args))
	}
	return &c.cmd
}

// Handle Error from real function
func funcError(err error) error {
	return err
}
