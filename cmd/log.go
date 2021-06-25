/**
 * This file going to implement the log subcomand which is responsible
 * for showing logs for acts running as daemons.
 */

package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/hpcloud/tail"
	"github.com/nosebit/act/run"
	"github.com/nosebit/act/utils"
)

//############################################################
// Global Variables
//############################################################
var ta *tail.Tail

//############################################################
// Exposed Functions
//############################################################

/**
 * This is the main execution point for the `log` command.
 */
func LogCmdExec(args []string) {
	/**
	 * We create a new flag set to allow this act subcommand to
	 * accepts flags by their own.
	 */
	cmdFlags := flag.NewFlagSet("log", flag.ExitOnError)

	/**
	 * This flag indicates we want to follow the logs as they
	 * are created. This is similar as tail shell command.
	 */
	followPtr := cmdFlags.Bool("f", false, "Follow file while it gets updated")

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
	 * For the log command we need user to provide one act name
	 * for the act we want to retrieve logs.
	 */
	if len(cmdArgs) < 1 {
		utils.FatalError("you need to specify the name of the act to log")
	}

	/**
	 * The first argument is the act name id.
	 *
	 * @TODO : Allow user to specify a list of act name ids so we
	 * can log everything together chronologically and tail
	 * all of them together. This can be usefule to see on act
	 * calling another act for example and tracing logs.
	 */
	actNameId := cmdArgs[0]

	/**
	 * Get act run info
	 */
	info := run.GetInfo(actNameId)

	if info == nil {
		utils.FatalError("act not found")
	}

	logFilePath := info.GetLogFilePath()

	if _, err := os.Stat(logFilePath); err != nil {
		utils.FatalError("nothing to log")
	}

	/**
	 * @TODO : For some reason logs are not being shown until we get
	 * enought logs to fulfill the offset. When we have few logs the
	 * tail package shows nothing.
	 */

	t, err := tail.TailFile(logFilePath, tail.Config{
		Follow: *followPtr,
		Location: &tail.SeekInfo{
			Offset: -500,
			Whence: 2, // 0 - Begining of file; 1 - Current Position; 2 - End of file
		},
		ReOpen: *followPtr,
		Logger: tail.DiscardingLogger,
	})

	// Store tail globally so we can cleanup
	ta = t

	if err != nil {
		utils.FatalError("could not open log file", err)
	}

	/**
	 * We prevent logging the first line because it could be
	 * broken since SeekInfo used before specifies number of
	 * bytes as offset.
	 *
	 * @TODO - It would be amazing if there was a way to let
	 * user specify the number of lines (from the end of file)
	 * to log before starting following the log file.
	 */
	isFirstLine := true

	for line := range t.Lines {
		if !isFirstLine {
			fmt.Println(line.Text)
		}

		isFirstLine = false
	}
}

/**
 * This function going to cleanup everything for this command on exit.
 */
func LogCleanup() {
	if ta != nil {
		ta.Cleanup()
		ta.Stop()
	}
}
