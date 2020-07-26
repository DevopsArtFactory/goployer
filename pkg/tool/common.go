package tool

import (
	"fmt"
	Logger "github.com/sirupsen/logrus"
	"log"
	"os"
	"reflect"
	"time"
)

var (
	NO_ERROR_MESSAGE_PASSED = "No Error Message exists"
	INITIAL_STATUS          = "Not Found"
	DAYTOSEC                = int64(86400)
	HOURTOSEC               = int64(3600)
)

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
		Red(NO_ERROR_MESSAGE_PASSED)
		os.Exit(1)
	}
	Red(msg)
	os.Exit(1)
}

// Fatal Error
func FatalError(err error) {
	log.Fatalf("error: %v", err)
	os.Exit(1)
}

func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool() == false

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0

	case reflect.Float32, reflect.Float64:
		return v.Float() == 0

	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0

	case reflect.Ptr, reflect.Interface:
		return isZero(v.Elem())

	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true

	case reflect.Slice, reflect.String, reflect.Map:
		return v.Len() == 0

	case reflect.Struct:
		for i, n := 0, v.NumField(); i < n; i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	// reflect.Chan, reflect.UnsafePointer, reflect.Func
	default:
		return v.IsNil()
	}
}

// IsZero reports whether v is zero struct
// Does not support cycle pointers for performance, so as json
func IsZero(v interface{}) bool {
	return isZero(reflect.ValueOf(v))
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
func CheckTimeout(start int64, timeout time.Duration) bool {
	now := time.Now().Unix()
	timeoutSec := int64(timeout / time.Second)

	//Over timeout
	if (now - start) > timeoutSec {
		Logger.Errorf("Timeout has been exceeded : %.0f minutes", timeout.Minutes())
		os.Exit(1)
	}

	return false
}

//Get KST Timestamp
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
