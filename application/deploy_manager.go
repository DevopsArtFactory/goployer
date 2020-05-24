package application

type DeployManager interface {
	Deploy(config Config)
	CleanPreviousVersion() error
	Healthchecking(config Config) map[string]bool
	TerminateChecking(config Config) map[string] bool
	GetStackName() string
}
