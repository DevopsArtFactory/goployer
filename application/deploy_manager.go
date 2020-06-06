package application

type DeployManager interface {
	GetStackName() string
	Deploy(config Config)
	HealthChecking(config Config) map[string]bool
	FinishAdditionalWork(config Config) error
	CleanPreviousVersion(config Config) error
	TerminateChecking(config Config) map[string] bool
}
