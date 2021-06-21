package utils

import (
	"log"
	"os"
)

//############################################################
// Internal Variables
//############################################################
var (
	errorLogger *log.Logger
	debugLogger *log.Logger
	infoLogger  *log.Logger
)

//############################################################
// Exposed Functions
//############################################################

/**
 * This function going to log an error.
 */
func LogError(args ...interface{}) {
	errorLogger.Println(args...)
}

/**
 * This function log debug messages.
 */
func LogDebug(args ...interface{}) {
	if _, present := os.LookupEnv("ACT_DEBUG"); present {
		debugLogger.Println(args...)
	}
}

/**
 * This function going to log an info message.
 */
func LogInfo(args ...interface{}) {
	infoLogger.Println(args...)
}

/**
 * This function going to handle fatal error.
 */
func FatalError(args ...interface{}) {
	LogError(args...)
	os.Exit(1)
}

//############################################################
// Lifecycle Functions
//############################################################

/**
 * On init we going to create all custom loggers.
 */
func init() {
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime)
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime)
}
