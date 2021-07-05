/**
 * WaitGroup example: https://adampresley.github.io/2015/02/16/waiting-for-goroutines-to-finish-running-before-exiting.html
 */

package main

import (
	"os"
	"os/signal"
	"sync"
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
// Internal Functions
//############################################################
func scheduleQuitCleanup() *sync.WaitGroup {
	var wg sync.WaitGroup

	/**
	 * Upon exit we going to clean up state.
	 */
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	/**
	 * Run our cleanup function as a go routine (i.e., in parallel) so
	 * we don't block the main execution since we need to wait for
	 * a quit event to do the cleanup job.
	 */
	go func() {
		/**
		 * This going to block the execution until sigs channel
		 * receive a quit signal.
		 */
		<-sigs

		/**
		 * Mark we are running a long running task that should be
		 * waited from the caller.
		 */
		wg.Add(1)

		/**
		 * Run cleanup functions for current executing command.
		 */
		cmd.Cleanup()

		/**
		 * Now we can safelly unblock the execution of the waiting
		 * caller.
		 */
		wg.Done()
	}()

	/**
	 * Returns the wait group so caller can wait (be blocked) until
	 * our cleanup go routine is done.
	 */
	return &wg
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
	 * We start by scheduling a cleanup job that going to be run
	 * on process termination. This cleanup schedule returns a
	 * wait group we going to wait at the end of this main function
	 * to allow cleanup finish correctly (which can take some time).
	 */
	cleanup := scheduleQuitCleanup()

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
	cmd.Exec(args)

	// Wait cleanup to finish
	cleanup.Wait()
}
