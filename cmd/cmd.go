/**
 * This is the main entrypoint for executing act cli commands.
 */
package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/nosebit/act/run"
)

//############################################################
// Internal Variables
//############################################################

/**
 * This is the name of the command we are currently executing.
 */
var cmdName string

//############################################################
// Exposed Variables
//############################################################
var BinVersion = "development"
var BinOS = ""
var BinArch = ""
var BinBuildTime = ""

//############################################################
// Exposed Functions
//############################################################
/**
 * This is the entrypoint function of this package and it's going to decide
 * which act cli command to run.
 */
func Exec(args []string) {
	cmdName = args[0]

	switch cmdName {
	case "version":
		binOS := BinOS
		binArch := BinArch

		if binOS == "" {
			binOS = runtime.GOOS
		}

		if binArch == "" {
			binArch = runtime.GOARCH
		}

		fmt.Printf("act version %s %s/%s %s\n", BinVersion, binOS, binArch, BinBuildTime)
	case "run":
		run.Exec(args[1:])
	case "log":
		LogCmdExec(args[1:])
	case "list":
		ListCmdExec()
	case "stop":
		StopCmdExec(args[1:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
}

/**
 * This function going to call the correct cleanup function
 * for the executing command.
 */
func Cleanup() {
	switch cmdName {
	case "run":
		run.Cleanup()
	case "log":
		LogCleanup()
	default:
	}
}
