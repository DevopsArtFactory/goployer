package builder

import (
	"os"
)


var (
	NO_ERROR_MESSAGE_PASSED="No Error Message exists"
)

// Check if file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Error Logging
func error_logging(msg string)  {
	if len(msg) == 0 {
		Red(NO_ERROR_MESSAGE_PASSED)
		os.Exit(1)
	}
	Red(msg)
	os.Exit(1)
}

