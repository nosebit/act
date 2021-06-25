/**
 * This file going to implement the list subcommand which
 * is responsible for listing all running acts.
 */

package cmd

import (
	"fmt"
	"os"

	"github.com/logrusorgru/aurora/v3"
	"github.com/nosebit/act/run"
	"github.com/olekukonko/tablewriter"
)

//############################################################
// Exposed Functions
//############################################################

/**
 * This is the main execution point for the `list` command.
 */
func ListCmdExec() {
	infos := run.GetAllInfo()

	if len(infos) == 0 {
		fmt.Println(aurora.Yellow("no act running").Bold())
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
  table.SetHeader([]string{"Id", "Name"})

	for _, info := range infos {
		table.Append([]string{info.Id, info.NameId})
	}

  table.Render()
}
