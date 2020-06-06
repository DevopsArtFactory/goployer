package application

type DeployManager interface {
	Deploy(config Config)
	CleanPreviousVersion() error
	HealthChecking(config Config) map[string]bool
	FinishAdditionalWork() error
	TerminateChecking(config Config) map[string] bool
	GetStackName() string
}
