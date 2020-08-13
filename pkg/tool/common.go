package tool

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	Logger "github.com/sirupsen/logrus"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	NO_ERROR_MESSAGE_PASSED = "No Error Message exists"
	INITIAL_STATUS          = "Not Found"
	DAYTOSEC                = int64(86400)
	HOURTOSEC               = int64(3600)
	allowedAnswerYes        = []string{"y", "yes"}
	allowedAnswerNo         = []string{"n", "no"}
	LogLevelMapper          = map[string]Logger.Level{
		"info":  Logger.InfoLevel,
		"debug": Logger.DebugLevel,
		"warn":  Logger.WarnLevel,
		"trace": Logger.TraceLevel,
		"fatal": Logger.FatalLevel,
		"error": Logger.ErrorLevel,
	}
	// STEP_CHECK_PREVIOUS = CheckPrevious
	STEP_CHECK_PREVIOUS = int64(1)
	// STEP_DEPLOY = Deploy
	STEP_DEPLOY = int64(2)
	// STEP_ADDITIONAL_WORK = FinishAdditionalWork
	STEP_ADDITIONAL_WORK = int64(3)
	// STEP_TRIGGER_LIFECYCLE_CALLBACK = TriggerLifecycleCallbacks
	STEP_TRIGGER_LIFECYCLE_CALLBACK = int64(4)
	// STEP_CLEAN_PREVIOUS_VERSION = CleanPreviousVersion
	STEP_CLEAN_PREVIOUS_VERSION = int64(5)
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// Check if file exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Error Logging
func ErrorLogging(msg string) {
	if len(msg) == 0 {
		Red.Fprintln(os.Stderr, NO_ERROR_MESSAGE_PASSED)
	} else {
		Red.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

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

//Check timeout
func CheckTimeout(start int64, timeout time.Duration) (bool, error) {
	now := time.Now().Unix()
	timeoutSec := int64(timeout / time.Second)

	//Over timeout
	if (now - start) > timeoutSec {
		return true, nil
	}

	return false, nil
}

func GetBaseTimeWithTimestamp(timezone string) time.Time {
	now := time.Now()

	loc, _ := time.LoadLocation(timezone)
	return now.In(loc)
}

func GetBaseTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func GetBaseStartTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location()).Add(1 * time.Hour)
}

func GetTimePrefix(t time.Time) string {
	return fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())
}

func AskContinue(message string) bool {
	var answer string
	prompt := &survey.Input{
		Message: message,
	}
	survey.AskOne(prompt, &answer)
	if answer == "" {
		return false
	}

	if IsStringInArray(answer, allowedAnswerYes) {
		return true
	}

	return false
}

func CheckFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
