package logger

import (
	"fmt"
	"log"
	"os"
)

var (
	debugLogger *log.Logger
	isDebug     bool
)

func Init(debug bool) {
	debugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	isDebug = debug
}

func Debug(format string, v ...interface{}) {
	if isDebug && debugLogger != nil {
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}
