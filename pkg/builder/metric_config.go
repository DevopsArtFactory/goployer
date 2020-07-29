package builder

type MetricBuilder struct { // Do not add comments for this struct
	MetricConfig MetricConfig
}

// Metric Builder Configurations
type MetricConfig struct {
	// Whether or not to gather metrics
	Enabled bool

	// Base region for gathering metrics
	Region string `yaml:"region"`

	// Storage configuration for storing metric data
	Storage Storage `yaml:"storage"`

	// Configuration of metrics
	Metrics Metrics `yaml:"metrics"`
}

// Storage configurations
type Storage struct {
	// Type of storage
	Type string `yaml:"type"`

	// Name of storage
	Name string `yaml:"name"`
}

// Configurations of metrics
type Metrics struct {
	// Timezone of metrics
	BaseTimezone string `yaml:"base_timezone"`
}
