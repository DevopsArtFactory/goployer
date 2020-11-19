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
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/AlecAivazis/survey/v2"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
)

// Fatal Error
func FatalError(err error) {
	log.Fatalf("error: %v", err)
	os.Exit(1)
}

// IsStringInArray checks if string value is in array or not
func IsStringInArray(s string, arr []string) bool {
	for _, as := range arr {
		if as == s {
			return true
		}
	}
	return false
}

// IsStringInPointerArray checks if string value is in array or not
func IsStringInPointerArray(s string, arr []*string) bool {
	for _, as := range arr {
		if *as == s {
			return true
		}
	}
	return false
}

//CheckTimeout compares now-start time with timeout
func CheckTimeout(start int64, timeout time.Duration) (bool, error) {
	now := time.Now().Unix()
	timeoutSec := int64(timeout / time.Second)

	//Over timeout
	if (now - start) > timeoutSec {
		return true, nil
	}

	return false, nil
}

// GetBaseTimeWithTimezone returns time with timezone
func GetBaseTimeWithTimezone(timezone string) time.Time {
	now := time.Now()

	loc, _ := time.LoadLocation(timezone)
	return now.In(loc)
}

// GetBaseTime generates base time format
func GetBaseTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// GetBaseStartTime generates start time
func GetBaseStartTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location()).Add(1 * time.Hour)
}

// GetTimePrefix returns time prefix
func GetTimePrefix(t time.Time) string {
	return fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())
}

// AskContinue asks a user whether or not to continue the process
func AskContinue(message string) bool {
	var answer string
	prompt := &survey.Input{
		Message: message,
	}
	survey.AskOne(prompt, &answer)
	if answer == "" {
		return false
	}

	if IsStringInArray(answer, constants.AllowedAnswerYes) {
		return true
	}

	return false
}

// CheckFileExists checks if a file or a directory exists or not
func CheckFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// IsTargetGroupArn returns true if string is target group ARN
func IsTargetGroupArn(str string, region string) bool {
	return strings.HasPrefix(str, fmt.Sprintf("arn:aws:elasticloadbalancing:%s", region)) && strings.Contains(str, "targetgroup")
}

// IsCanaryTargetGroupArn returns true if string is target group ARN
func IsCanaryTargetGroupArn(str string, region string) bool {
	return strings.HasPrefix(str, fmt.Sprintf("arn:aws:elasticloadbalancing:%s", region)) && strings.Contains(str, "targetgroup") && strings.Contains(str, constants.CanaryMark)
}

// RoundTime creates rounded time
func RoundTime(d time.Duration) string {
	var r float64
	var suffix string
	switch {
	case d > time.Minute:
		r = d.Minutes()
		suffix = "m"
	case d > time.Second:
		r = d.Seconds()
		suffix = "s"
	default:
		r = float64(d.Milliseconds())
		suffix = "ms"
	}

	return fmt.Sprintf("%.2f%s", r, suffix)
}

// RoundNum create rounded number
func RoundNum(n float64) string {
	return fmt.Sprintf("%.2f", n)
}

// JoinString joins strings in the slice
func JoinString(arr []string, delimiter string) string {
	return strings.Join(arr, delimiter)
}

// CreateBodyStruct creates body with slice
func CreateBodyStruct(slice []string) ([]byte, error) {
	bd := map[string]string{}
	for _, s := range slice {
		split := strings.Split(s, "=")
		bd[split[0]] = split[1]
	}

	jsonBody, err := json.Marshal(bd)
	if err != nil {
		return nil, err
	}

	return jsonBody, nil
}

// CreateHeaderStruct creates header with slice
func CreateHeaderStruct(slice []string) (http.Header, error) {
	hd := SetCommonHeader()
	for _, s := range slice {
		split := strings.Split(s, "=")
		if len(split) != 2 {
			return nil, fmt.Errorf("wrong format header: %s", s)
		}
		hd.Add(split[0], split[1])
	}

	return hd, nil
}

// SetCommonHeader returns common header for api test
func SetCommonHeader() http.Header {
	return http.Header{
		"Content-Type": []string{"application/json"},
	}
}

// ParseTargetGroupName parses target group ARN and return target group name
func ParseTargetGroupName(arn string) string {
	return strings.Split(arn, "/")[1]
}

// LocalCheck checks whether or not to continue when it is run on localhost.
// Cannot add windows because goployer could be run on Windows..
func LocalCheck(message string, autoApply bool) error {
	// From local os, you need to ensure that this command is intended
	if runtime.GOOS == "darwin" && !autoApply {
		if !AskContinue(message) {
			return errors.New("you declined to run command")
		}
	}
	return nil
}

// PrintTemplate prints template with data
func PrintTemplate(data interface{}, t *template.Template) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 5, 3, ' ', tabwriter.TabIndent)
	err := t.Execute(w, data)
	if err != nil {
		return err
	}
	return w.Flush()
}
