package run

import (
	"fmt"
	//"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/nosebit/act/actfile"
	"github.com/nosebit/act/utils"
	"github.com/teris-io/shortid"
)

//############################################################
// Internal Functions
//############################################################

/**
 * This function going to run an act in detached mode. In this
 * mode the act going to be run as separate act process which
 * can be managed independently (stopped/logged).
 */
func ActDetachExec(cmd *actfile.Cmd, ctx *ActRunCtx, wg *sync.WaitGroup) {
	actFilePath := ctx.ActFile.LocationPath

	if cmd.From != "" {
		actFilePath = utils.ResolvePath(path.Dir(ctx.ActFile.LocationPath), cmd.From)
	}

	childId, _ := shortid.Generate()

	envars := []string {
		fmt.Sprintf("ACT_PARENT_RUN_ID=%s", ctx.RunCtx.Info.Id),
		fmt.Sprintf("ACT_RUN_ID=%s", childId),
	}

	// Set environment vars
	vars := ctx.MergeVars()
	envars = append(envars, utils.VarsMapToEnvVars(vars)...)

	actNameId := utils.CompileTemplate(cmd.Act, vars)
	cmdLineArgs := []string{"run", fmt.Sprintf("-f=%s", actFilePath), actNameId}
	cmdLineArgs = append(cmdLineArgs, cmd.Args...)

	shCmd := exec.Command("act", cmdLineArgs...)
	shCmd.Dir = utils.GetWd()
	shCmd.Env = append(os.Environ(), envars...)
	shCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set logging
	if !ctx.RunCtx.Quiet && !ctx.Act.Quiet && !cmd.Quiet {
		l := NewLogWriter(ctx)

		/**
		 * For detached processes we going to pevent logging prefix
		 * info on this parent process so we don't end up having
		 * double prefix infos. The prefixing going to be done
		 * in the child process itself and here we just log whatever
		 * child process send to us (prefixed).
		 */
		l.Detached = true

		shCmd.Stdout = l
		shCmd.Stderr = l
	}

	// Start act execution
	shCmd.Start()

	// Add child id
	ctx.RunCtx.Info.AddChildId(childId)

	// Wait child process finalization.
	shCmd.Wait()

	if wg != nil {
		wg.Done()
	}
}

//############################################################
// Exported Functions
//############################################################

/**
 * This function execute multiple commands withing a specific
 * act run context.
 */
func CmdsExec(cmds []*actfile.Cmd, ctx *ActRunCtx, wg *sync.WaitGroup) {
	for _, cmd := range cmds {
		if ctx.Act.Parallel {
			go CmdExec(cmd, ctx, wg)
		} else {
			CmdExec(cmd, ctx, nil)
		}
	}
}

/**
 * This function going to execute a command.
 */
func CmdExec(cmd *actfile.Cmd, ctx *ActRunCtx, wg *sync.WaitGroup) {
	if ctx.RunCtx.IsKilling {
		return
	}

	/**
	 * Merge all local vars together respecting overide rules.
	 */
	vars := ctx.MergeVars()

	/**
	 * If command specify a loop then we going to execute multiple
	 * generated commands.
	 */
	if cmd.Loop != nil {
		var items []string

		if cmd.Loop.Glob != "" {
			glob := utils.CompileTemplate(cmd.Loop.Glob, vars)
			pattern := utils.ResolvePath(utils.GetWd(), glob)
			paths, err :=  filepath.Glob(pattern)

			if err != nil {
				utils.FatalError("glob error", err)
			}

			items = paths
		} else {
			items = cmd.Loop.Items
		}

		if len(items) > 0 {
			var cmds []*actfile.Cmd

			for _, item := range items {
				vars["LoopItem"] = item

				genCmd := actfile.Cmd{
					Cmd: utils.CompileTemplate(cmd.Cmd, vars),
					Act: utils.CompileTemplate(cmd.Act, vars),
					From: utils.CompileTemplate(cmd.From, vars),
					Args: cmd.Args,
					Script: cmd.Script,
					Detach: cmd.Detach,
					Mismatch: cmd.Mismatch,
					Quiet: cmd.Quiet,
				}

				cmds = append(cmds, &genCmd)
			}

			CmdsExec(cmds, ctx, wg)
		}

		return
	}

	/**
	 * If command is invoking another act then lets run it.
	 */
	if cmd.Act != "" {

		/**
		 * If we want to run the act as separate act process
		 * (detached mode) then let's spawn the process.
		 */
		if cmd.Detach {
			ActDetachExec(cmd, ctx, wg)
			return
		}

		actField := utils.CompileTemplate(cmd.Act, vars)
		actNames := strings.Split(actField, ActCallIdSeparator)
		actFile := ctx.ActFile

		// Set actfile to look up for act.
		if cmd.From != "" {
			from := utils.CompileTemplate(cmd.From, vars)
			actFilePath := utils.ResolvePath(utils.GetWd(), from)

			if actFile.LocationPath != actFilePath {
				actFile = actfile.ReadActFile(actFilePath)
			}
		}

		nextCtx, err := FindActCtx(actNames, actFile, ctx, ctx.RunCtx)

		if err != nil {
			/**
			 * If we didn't found an act to run but we allow mismatch
			 * then lets just skip the not found act with no errors.
			 * This is useful when invoking acts from a list of generic
			 * actfiles located in subfolders.
			 */
			if cmd.Mismatch == "allow" {
				return
			}

			utils.FatalError(err)
		}

		nextCtx.Args = cmd.Args
		nextCtx.Act.Log = ctx.Act.Log

		nextCtx.Exec()
		return
	}

	/**
	 * Set the command to run (script or command line).
	 */
	var shArgs []string

	if cmd.Script != "" {
		cmdLine := utils.CompileTemplate(cmd.Script, vars)

		shArgs = append([]string{cmdLine}, ctx.Args...)
	} else {
		cmdLine := utils.CompileTemplate(cmd.Cmd, vars)

		shArgs = []string{"-c", cmdLine, "--"}
		shArgs = append(shArgs, ctx.Args...)
	}
	
	// Set shell to use in the right precedence order.
	shell := "bash"

	if ctx.ActFile.Shell != "" {
		shell = ctx.ActFile.Shell
	}

	if ctx.Act.Shell {
		shell = ctx.Act.Shell
	}

	if cmd.Shell != "" {
		shell = cmd.Shell
	}

	// Command to spawn.
	shCmd := exec.Command(shell, shArgs...)

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
	 * Set environment variables using all available variables.
	 */
	envars := utils.VarsMapToEnvVars(vars)

	/**
	 * Set a special ACT_ENV_FILE variable pointing to the full
	 * path to env file set on actfile.
	 */
	if ctx.ActFile.EnvFilePath != "" {
		envFilePath := utils.ResolvePath(path.Dir(ctx.ActFile.LocationPath), ctx.ActFile.EnvFilePath)

		envars = append(envars, fmt.Sprintf("ACT_ENV_FILE=%s", envFilePath))
	}

	// Set all env vars to shell command.
	shCmd.Env = append(os.Environ(), envars...)

	/**
	 * Set output
	 */
	if !ctx.RunCtx.Quiet && !ctx.Act.Quiet && !cmd.Quiet {

		/**
		 * Set the log mode. By default log mode is `raw` and therefore we going
		 * to send all logs directly to stdout without any prefixing containing
		 * act info. If we want to prepend log lines with a prefix containing
		 * act name id and timestamp we can set log mode as `prefixed`.
		 */
		logMode := "raw"

		if ctx.ActFile.Log != "" {
			logMode = ctx.ActFile.Log
		}

		if ctx.Act.Log != "" {
			logMode = ctx.Act.Log
		}

		if ctx.RunCtx.Log != "" {
			logMode = ctx.RunCtx.Log
		}

		if !ctx.RunCtx.IsDaemon && logMode == "raw" {
			shCmd.Stdout = os.Stdout
			shCmd.Stderr = os.Stderr
		} else {
			/**
			 * Log writer going to log output with a prefix containing
			 * act name id and timestamp both to stdout and to a log file.
			 * If the spawn process log output with color it probably going
			 * to lose colors here (like jest logging).
			 */
			l := NewLogWriter(ctx)

			shCmd.Stdout = l
			shCmd.Stderr = l
		}
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
	shCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
	ctx.RunCtx.Info.AddChildPgid(pgid)

	// Wait finalization
	shCmd.Wait()

	// Remove pgid now
	if !ctx.RunCtx.IsKilling {
		ctx.RunCtx.Info.RmChildPgid(pgid)
	}

	if wg != nil {
		wg.Done()
	}
}
