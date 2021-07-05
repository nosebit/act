/**
 * This program is just and alias for `act run` command.
 */
package main

import (
	"os"
	"os/exec"
)

/**
 * Note: By convention the entrypoint file of which package
 * going to be named the same as the package itself. So for
 * example the entrypoint file of actfile package going to
 * be actfile/actfile.go.
 */

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
	cmd := exec.Command("act", args...)

	// Set all env vars to shell command.
	cmd.Env = os.Environ()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start and wait
	cmd.Run()
}
