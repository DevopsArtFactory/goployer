package application

import (
	"fmt"
	"strings"
	"strconv"
	"time"
)

var (
	COUNTRIES = 'c'
	DEV_PHASE = 'd'
	HARDWARE = 'h'
	PARTNERS = 'p'
	REVISION = 'r'
	USED_BY = 'u'
	RED_BLACK_SWAP = 'w'
	ZONE = 'z'
	VERSION = 'v'
)

type Frigga struct {
	Prefix string
}

func buildPrefixName(name string, env string, region string) string {
	return fmt.Sprintf("%s-%s_%s", name, env, strings.ReplaceAll(region, "-", ""))
}

func parseVersion(name string) int {
	if len(name) != 0 {
		parts := strings.Split(name, "-")
		for _, part := range parts {
			if len(part) > 0 && strings.HasPrefix(part, "v"){
				intVal, _ :=  strconv.Atoi(part[1:])
				return intVal
			}
		}
	}

	return 0
}

// generateAsgName generates the autoscaling name
func generateAsgName(prefix string, version int) string {
	return fmt.Sprintf("%s-v%03d", prefix, version)
}

// generateLcName generates new launch configuration name
func generateLcName(asg_name string) string {
	now := time.Now()
	secs := now.Unix()
	return fmt.Sprintf("%s-%d", asg_name, secs)
}
