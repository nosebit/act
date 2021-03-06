/**
 * This file implements the stop subcommand which is responsible
 * for stopping acts running in the background as daemon.
 */

package cmd

import (
	"flag"

	"github.com/nosebit/act/cmd/act/run"
	"github.com/nosebit/act/cmd/act/utils"
)

//############################################################
// Exposed Functions
//############################################################

/**
 * This is the main execution point for the `log` command.
 */
func StopCmdExec(args []string) {
	/**
	 * We create a new flag set to allow this act subcommand to
	 * accepts flags by their own.
	 */
	cmdFlags := flag.NewFlagSet("stop", flag.ExitOnError)

	/**
	 * Parse the incoming args extracting defined flags if user
	 * provided any.
	 */
	cmdFlags.Parse(args)

	/**
	 * This are the command line arguments after extracting
	 * the flags.
	 */
	cmdArgs := cmdFlags.Args()

	/**
	 * For the stop command we need user to provide an act name
	 * id for the act which going to be stopped.
	 */
	if len(cmdArgs) < 1 {
		utils.FatalError("you need to specify the name of the act to stop")
	}

	/**
	 * The first argument is the act name id we want to stop.
	 *
	 * @TODO : Allow users to provide a list of act name ids to
	 * stop everything together and maybe provide a stop all
	 * by running something like `act stop *`.
	 */
	actNameId := cmdArgs[0]

	// Get act info
	info := run.GetInfo(actNameId)

	if info == nil {
		utils.FatalError("act not found")
	}

	// Kill it
	info.Kill()
}
