/**
 * WaitGroup example: https://adampresley.github.io/2015/02/16/waiting-for-goroutines-to-finish-running-before-exiting.html
 */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	/**
	 * Learning Note: Different packages need to be imported
	 * so we can use functions/properties exposed on those
	 * packeges.
	 */

	"github.com/nosebit/act/cmd/act/cmd"
	"github.com/nosebit/act/cmd/act/utils"
)

/**
 * Note: By convention the entrypoint file of which package
 * going to be named the same as the package itself. So for
 * example the entrypoint file of actfile package going to
 * be actfile/actfile.go.
 */

//############################################################
// Internal Variables
//############################################################
var killed bool

//############################################################
// Internal Functions
//############################################################
func scheduleStopOnKill() {
	/**
	 * Upon exit we going to clean up state.
	 */
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	/**
	 * When we receive a kill process we going to stop the current
	 * execution.
	 */
	go func() {
		/**
		 * This going to block the execution until sigs channel
		 * receive a quit signal.
		 */
		<-sigs

		utils.LogDebug("Received kill signal")

		/**
		 * Skip one line to prevent showing `^C` in the terminal
		 * next to logs for final commands like the following:
		 *
		 * ```text
		 * hello long1
		 * hello long2
		 * hello long2
		 * hello long1
		 * hello long2
		 * ^Ccleaning 1
		 * cleaning 2
		 * cleaning 3
		 * cleaning 4
		 * ```
		 */
		fmt.Println()

		killed = true

		/**
		 * Stop execution.
		 */
		cmd.Stop();
	}()
}

//############################################################
// Main Entrypoint
//############################################################
/**
 * This is the entrypoint function go going to call to start
 * our app. Everything starts here and therefore we start by
 * parsing actfile in current working directory and then
 * check what command we are invoking.
 */
func main() {

	/**
	 * We start by scheduling a stop job that going to be run
	 * on process termination. This way when client sends a kill
	 * signal we going to stop the main execution.
	 */
	scheduleStopOnKill()

	//--------------------------------------------------
	// Parse command line args
	//--------------------------------------------------
	args := os.Args[1:]

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main act command name
	// os.Arg[1] is act subcommand
	if len(args) < 1 {
		utils.FatalError("subcommand is required")
	}

	// Now we execute subcommand (synchronously)
	/**
	 * Execute the subcomand. The execution of all commands
	 * going to be synchronously and therefore going to block
	 * this main thread while execution is running. When the
	 * process receive a Kill signal it going first to stop
	 * the main execution which going to make cmd.Exec to return
	 * the control to this main flow here.
	 */
	cmd.Exec(args)

	/**
	 * Now that main execution is done (or stopped because of a kill)
	 * we going to run the finish stage so we can gracefully exit.
	 * In the finish stage we going to run any final commands the
	 * cliend defined in the actfile to cleanup the execution before
	 * exiting.
	 */
	cmd.Finish()

	// Now exit with correct exit code.
	os.Exit(utils.ExitCode)
}
