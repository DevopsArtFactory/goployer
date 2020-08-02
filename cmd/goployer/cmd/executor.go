package cmd

import "context"

// Run function without executor
func runWithoutExecutor(ctx context.Context, action func() error) error {
	err := action()

	return alwaysSucceedWhenCancelled(ctx, err)
}
