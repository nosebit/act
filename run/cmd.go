package run

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/nosebit/act/actfile"
	"github.com/nosebit/act/utils"
)

//############################################################
// Exported Functions
//############################################################

/**
 * This function going to execute a command.
 */
func CmdExec(cmd *actfile.Cmd, ctx *ActRunCtx, wg *sync.WaitGroup) {
	if ctx.RunCtx.IsKilling {
		return
	}

	/**
	 * If command is invoking another act then lets run it.
	 */
	if cmd.Act != "" {
		// Find next act ctx to run
		actNames := strings.Split(cmd.Act, ActCallIdSeparator)

		nextCtx := FindActCtx(actNames, ctx.ActFile, ctx, ctx.RunCtx)
		nextCtx.Args = cmd.Args

		nextCtx.Exec()
		return
	}

	/**
	 * Merge all local vars together.
	 */
	vars := make(map[string]string)

	for key, val := range ctx.RunCtx.Vars {
		vars[key] = val
	}

	for key, val := range ctx.Vars {
		vars[key] = val
	}

	for key, val := range ctx.FlagVals {
		vars[key] = val
	}

	// Compile command line
	cmdLine := utils.CompileTemplate(cmd.Cmd, vars)
	cmdLineParts := strings.Split(cmdLine, " ")

	shArgs := []string{"-c", cmdLine, "--"}
	shArgs = append(shArgs, ctx.Args...)

	shCmd := exec.Command("bash", shArgs...)

	/**
	 * We going to run the scrip relative to the folder which contains
	 * the actfile where we actually matched the act to run.
	 */
	shCmd.Dir = path.Dir(ctx.ActFile.LocationPath)

	/**
	 * Load env vars
	 */
	godotenv.Load(ctx.RunCtx.Info.GetEnvVarsFilePath())

	/**
	 * Set environment variables using all available env vars plus
	 * env vars built from local vars.
	 */
	envars := utils.VarsMapToEnvVars(vars)

	/**
	 * Set log prefix.
	 */
	logPrefix := ctx.RunCtx.Info.NameId

	if ctx.ActFile.Namespace != "" {
		logPrefix = fmt.Sprintf("%s.%s", ctx.ActFile.Namespace, ctx.Act.Name)
	}

	/**
	 * If we are invoking act in the command we going to set an
	 * env variable to adjust the logs for side running acts.
	 */
	var parentActPrefix string

	if cmdLineParts[0] == "act" {
		parentActList := []string{logPrefix}

		if parent, present := os.LookupEnv("ACT_PARENT_ACT"); present {
			parentActList = append([]string{parent}, logPrefix)
		}

		parentActPrefix = strings.Join(parentActList, " > ")

		envars = append(envars, fmt.Sprintf("ACT_PARENT_ACT=%s", parentActPrefix))
	}

	shCmd.Env = append(os.Environ(), envars...)

	/**
	 * Set output
	 */
	if !ctx.RunCtx.Quiet && !ctx.Act.Quiet && !cmd.Quiet {
		prefix := logPrefix

		/**
		 * If we going to have a side act running as separate
		 * process then we going to prevent any prefix at all.
		 */
		if parentActPrefix != "" {
			prefix = ""
		}

		shCmd.Stdout = NewLogWriter(prefix, false)
		shCmd.Stderr = NewLogWriter(prefix, true)
	}

	/**
	 * We ask go to create a new process group for the command we
	 * going to execute. With that we can safelly kill all process
	 * group (i.e., all descendent process) withou killing the main
	 * act process which is spawning the command. If we don't ask go
	 * to create a fresh process group then by default the spawned
	 * process going to be assigned to the same process group as the
	 * main act process and if we kill it we going to commit suicide.
	 *
	 * Further explanations in:
	 *
	 * https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
	 */
	if _, present := os.LookupEnv("ACT_DAEMON_ID"); !present {
		shCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	// Start act execution
	shCmd.Start()

	/**
	 * Now that act is executing we can collect some runtime info like
	 * process id, etc.
	 */
	pid := shCmd.Process.Pid

	/**
	 * Try to get process group id so we can kill all child processes.
	 */
	pgid, err := syscall.Getpgid(pid)

	if err != nil {
		utils.FatalError(fmt.Sprintf("could not get pgid for pid=%d", pid), err)
	}

	ctx.Pgids = append(ctx.Pgids, pgid)

	// Save to run context info file
	ctx.RunCtx.Info.AddPgid(pgid)

	// Wait finalization
	shCmd.Wait()

	// Remove pgid now
	if !ctx.RunCtx.IsKilling {
		ctx.RunCtx.Info.RmPgid(pgid)
	}

	if wg != nil {
		wg.Done()
	}
}
