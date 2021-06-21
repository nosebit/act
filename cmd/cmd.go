/**
 * This is the main entrypoint for executing act cli commands.
 */
package cmd

import (
	"flag"
	"os"

	"github.com/nosebit/act/actfile"
)

//############################################################
// Internal Variables
//############################################################

/**
 * This is the name of the command we are currently executing.
 */
var cmdName string

//############################################################
// Exposed Functions
//############################################################
/**
 * This is the entrypoint function of this package and it's going to decide
 * which act cli command to run.
 */
func Exec(args []string, actfile *actfile.ActFile) {
	cmdName = args[0]

	switch cmdName {
	case "run":
		RunCmdExec(args[1:], actfile)
	case "log":
		LogCmdExec(args[1:], actfile)
	case "list":
		ListCmdExec(args[1:], actfile)
	case "stop":
		StopCmdExec(args[1:], actfile)
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
		RunCleanup()
	case "log":
		LogCleanup()
	default:
	}
}