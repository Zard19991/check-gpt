package logger

import (
	"fmt"
	"log"
	"os"
)

var (
	debugMode   bool
	debugLogger *log.Logger
)

// Init initializes the logger with debug mode
func Init(debug bool) {
	debugMode = debug
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Debug prints debug messages if debug mode is enabled
func Debug(format string, v ...interface{}) {
	if debugMode {
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}
