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
	Metrics Metrics
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
