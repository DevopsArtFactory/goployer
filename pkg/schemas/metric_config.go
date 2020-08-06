package schemas

type MetricBuilder struct { // Do not add comments for this struct
	MetricConfig MetricConfig
}

// Metric Builder Configurations
type MetricConfig struct {
	// Whether or not to gather metrics
	Enabled bool

	// Base region for gathering metrics
	Region string `yaml:"region"`

	//  Configuration for storage
	Storage Storage `yaml:"storage"`

	// Configuration of metrics
	Metrics Metrics `,inline`
}

// Storage configurations
type Storage struct {
	// Storage Type - dynamodb
	Type string `yaml:"type"`

	// Storage Name
	Name string `yaml:"name"`
}

// Configurations of metrics
type Metrics struct {
	// Timezone of metrics
	BaseTimezone string
}
