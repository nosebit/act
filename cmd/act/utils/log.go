package utils

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/logrusorgru/aurora/v3"
)

//############################################################
// Internal Variables
//############################################################
var supressErrors bool = false

var (
	errorLogger *log.Logger
	debugLogger *log.Logger
	infoLogger  *log.Logger
)

//############################################################
// Exposed Variables
//############################################################
var ExitCode int = 0
var KillInProgress bool

//############################################################
// Internal Functions
//############################################################
/**
 * This function going to send a signal to current process to
 * exit gracefully.
 */
func exitGracefully() {
	if KillInProgress {
		return
	}

	KillInProgress = true
	pid := os.Getegid()

	// Send kill signal.
	syscall.Kill(pid, syscall.SIGQUIT)
}

//############################################################
// Exposed Functions
//############################################################

/**
 * This function going to silence logs.
 */
func SetSupressErrors() {
	supressErrors = true
}

/**
 * This function going to log an error.
 */
func LogError(args ...interface{}) {
	if !supressErrors {
		errorLogger.Println(args...)
	}
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

	if len(args) > 0 {
		LogError(args...)
	}

	ExitCode = 1
	exitGracefully()
}

/**
 * This function going to handle fatal error with code.
 */
func FatalErrorWithCode(code int, args ...interface{}) {
	LogError(args...)
	ExitCode = code
	exitGracefully()
}

//############################################################
// Lifecycle Functions
//############################################################

/**
 * On init we going to create all custom loggers.
 */
func init() {
	errorLogger = log.New(os.Stderr, fmt.Sprintf("%s", aurora.Red("[ERROR] ").Bold()), log.Ldate|log.Ltime)
	debugLogger = log.New(os.Stdout, fmt.Sprintf("%s", aurora.Gray(8-1, "[DEBUG] ").Bold()), log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger = log.New(os.Stdout, fmt.Sprintf("%s", aurora.Cyan("[INFO] ").Bold()), log.Ldate|log.Ltime)
}
