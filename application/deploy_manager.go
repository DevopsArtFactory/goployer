package application

type DeployManager interface {
	Deploy(config Config)
	Healthchecking(config Config) bool
	GetStackName() string
}
