/**
 * This program is just and alias for `act run` command.
 */
package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var cmd *exec.Cmd

/**
 * Note: By convention the entrypoint file of which package
 * going to be named the same as the package itself. So for
 * example the entrypoint file of actfile package going to
 * be actfile/actfile.go.
 */

func scheduleQuitCleanup() {
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

		// Wait command to gracefully exit.
		cmd.Wait()
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
	args := []string{"run"}

	args = append(args, os.Args[1:]...)

	// Command to spawn.
	cmd = exec.Command("act", args...)

	scheduleQuitCleanup()

	// Set all env vars to shell command.
	cmd.Env = os.Environ()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start and wait
	cmd.Run()
}
