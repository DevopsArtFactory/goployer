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

package tool

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Frigga struct {
	Prefix string
}

// BuildPrefixName creates new prefix for autoscaling group
func BuildPrefixName(name string, env string, region string) string {
	return fmt.Sprintf("%s-%s_%s", name, env, strings.ReplaceAll(region, "-", ""))
}

// ParseAutoScalingVersion parses autoscaling version from name
func ParseAutoScalingVersion(name string) int {
	if len(name) != 0 {
		parts := strings.Split(name, "-")
		for _, part := range parts {
			if len(part) > 0 && strings.HasPrefix(part, "v") {
				intVal, _ := strconv.Atoi(part[1:])
				return intVal
			}
		}
	}

	return 0
}

// GenerateAsgName generates the autoscaling name
func GenerateAsgName(prefix string, version int) string {
	return fmt.Sprintf("%s-v%03d", prefix, version)
}

// GenerateLcName generates new launch configuration name
func GenerateLcName(asgName string) string {
	now := time.Now()
	secs := now.Unix()
	return fmt.Sprintf("%s-%d", asgName, secs)
}

// ParseTargetGroupVersion parses autoscaling version from name
func ParseTargetGroupVersion(name string) int {
	if len(name) != 0 {
		parts := strings.Split(name, "-")
		if len(parts[len(parts)-1]) > 0 && strings.HasPrefix(parts[len(parts)-1], "v") {
			intVal, _ := strconv.Atoi(parts[len(parts)-1][1:])
			return intVal
		}
	}

	return 0
}
