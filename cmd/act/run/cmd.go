package run

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/nosebit/act/cmd/act/actfile"
	"github.com/nosebit/act/cmd/act/utils"
	"github.com/teris-io/shortid"
)

//############################################################
// Internal Functions
//############################################################

/**
 * This function get log mode.
 */
func getLogMode(_ *actfile.Cmd, ctx *ActRunCtx) string {
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

	return logMode
}

/**
 * This function going to run an act in detached mode. In this
 * mode the act going to be run as separate act process which
 * can be managed independently (stopped/logged).
 */
func actDetachExec(cmd *actfile.Cmd, ctx *ActRunCtx, wg *sync.WaitGroup) {
	actFilePath := ctx.ActFile.LocationPath

	if cmd.From != "" {
		actFilePath = utils.ResolvePath(path.Dir(ctx.ActFile.LocationPath), cmd.From)
	}

	childId, _ := shortid.Generate()

	utils.LogDebug("actDetachExec", childId)

	// Set environment vars
	vars := ctx.MergeVars()

	// Set some custom vars
	vars["ACT_PARENT_RUN_ID"] = ctx.RunCtx.Info.Id
	vars["ACT_RUN_ID"] = childId

	// Create env vars
	envars := ctx.VarsToEnvVars(vars)

	logMode := getLogMode(cmd, ctx)

	actNameId := utils.CompileTemplate(cmd.Act, vars)
	cmdLineArgs := []string{"run", fmt.Sprintf("-f=%s", actFilePath), fmt.Sprintf("-l=%s", logMode), actNameId}
	cmdLineArgs = append(cmdLineArgs, cmd.Args...)

	shCmd := exec.Command("act", cmdLineArgs...)
	shCmd.Dir = utils.GetWd()
	shCmd.Env = envars

	utils.LogDebug("actDetachExec : envars", envars)

	// Ensure we create a new session for the created process (this mean a new pgid).
	shCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	/**
	 * Detached acts going to log only to file. If user want to see logs
	 * he/she need to use the log command.
	 */
	//logFilePath := ctx.RunCtx.Info.GetLogFilePath()
	//logFile, _ := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	//shCmd.Stdout = logFile
	//shCmd.Stderr = logFile

	l := NewLogWriter(ctx)
	l.Detached = true
	l.LogToConsole = cmd.Log

	shCmd.Stdout = l
	shCmd.Stderr = l

	// Start act execution
	shCmd.Start()
	
	pid := shCmd.Process.Pid
	pgid, _ := syscall.Getpgid(pid)

	utils.LogDebug("actDetachExec : child act started", pid, pgid)

	// Add child id
	ctx.RunCtx.Info.AddChildActId(childId)

	utils.LogDebug("actDetachExec : done")

	if wg != nil {
		wg.Done()
	}
}

//############################################################
// Exported Functions
//############################################################

/**
 * This function going to execute a stage.
 */
func StageCmdsExec(stage *actfile.ActExecStage, ctx *ActRunCtx) {
	utils.LogDebug("StageCmdsExec", stage.Name, ctx.RunCtx.State)

	/**
	 * Prevent execution if we are not in the running state. This is
	 * important so we don't execute stages when we get killed by
	 * client (which is going to put the execution in the stopped state).
	 */
	if ctx.RunCtx.State != ExecStateRunning {
		return
	}

	ctx.CurrentStage = stage

	utils.LogDebug(fmt.Sprintf("StageCmdsExec : start execution [act=%s] [stage=%s] [cmds_count=%d]", ctx.Act.Name, stage.Name, len(stage.Cmds)))

	wg := sync.WaitGroup{}
	wg.Add(len(stage.Cmds))

	for idx, cmd := range stage.Cmds {
		/**
		 * Prevent keep executing this stage if we are not in the running state. This is
	   * important so we don't execute more commands when we get killed by
	   * client (which is going to put the execution in the stopped state).
		 */
		if ctx.RunCtx.State != ExecStateRunning {
			wg.Done()
			continue
		}

		utils.LogDebug(fmt.Sprintf("StageCmdsExec : cmd execution [act=%s] [stage=%s] [progress=%d/%d]", ctx.Act.Name, stage.Name, idx+1, len(stage.Cmds)))

		if stage.Parallel{
			go CmdExec(cmd, ctx, &wg)
		} else {
			CmdExec(cmd, ctx, &wg)
		}

		utils.LogDebug(fmt.Sprintf("StageCmdsExec : cmd execution done [act=%s] [stage=%s] [progress=%d/%d]", ctx.Act.Name, stage.Name, idx+1, len(stage.Cmds)))
	}

	// Wait execution of all commands.
	wg.Wait()
}

/**
 * This function going to execute a command.
 */
func CmdExec(cmd *actfile.Cmd, ctx *ActRunCtx, wg *sync.WaitGroup) {
	/**
	 * Prevent execution if we are not in the running state. This is
	 * important so we don't execute stages when we get killed by
	 * client (which is going to put the execution in the stopped state).
	 */
	if ctx.RunCtx.State != ExecStateRunning {
		return
	}

	utils.LogDebug(fmt.Sprintf("CmdExec : begin [act=%s]", ctx.Act.Name))

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
			baseDir := path.Dir(ctx.ActFile.LocationPath)
			glob := utils.CompileTemplate(cmd.Loop.Glob, vars)
			pattern := utils.ResolvePath(baseDir, glob)
			paths, err := filepath.Glob(pattern)

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
					Cmd:      utils.CompileTemplate(cmd.Cmd, vars),
					Act:      utils.CompileTemplate(cmd.Act, vars),
					From:     utils.CompileTemplate(cmd.From, vars),
					Args:     cmd.Args,
					Script:   cmd.Script,
					Detach:   cmd.Detach,
					Mismatch: cmd.Mismatch,
					Quiet:    cmd.Quiet,
				}

				cmds = append(cmds, &genCmd)
			}

			stage := &actfile.ActExecStage{
				Cmds:     cmds,
				Parallel: ctx.CurrentStage.Parallel,
			}

			StageCmdsExec(stage, ctx)
		}

		/**
		 * Now that we finished running the command we need to
		 * release the wait group (i.e., mark it as done).
		 */
		if wg != nil {
			wg.Done()
		}

		return
	}

	/**
	 * If command is invoking another act then lets run it.
	 */
	if cmd.Act != "" {
		utils.LogDebug(fmt.Sprintf("CmdExec : sub act found [act=%s]", ctx.Act.Name))

		/**
		 * If we want to run the act as separate act process
		 * (detached mode) then let's spawn the process.
		 */
		if cmd.Detach {
			actDetachExec(cmd, ctx, wg)
			return
		}

		actField := utils.CompileTemplate(cmd.Act, vars)
		actNames := strings.Split(actField, ActCallIdSeparator)
		actFile := ctx.ActFile
		var cmdArgs []string

		// Set actfile to look up for act.
		if cmd.From != "" {
			from := utils.CompileTemplate(cmd.From, vars)
			actFilePath := utils.ResolvePath(utils.GetWd(), from)

			if actFile.LocationPath != actFilePath {
				actFile = actfile.ReadActFile(actFilePath)
			}
		}

		for _, arg := range cmd.Args {
			compiledArg := utils.CompileTemplate(arg, vars)
			cmdArgs = append(cmdArgs, compiledArg)
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

		nextCtx.Args = cmdArgs
		nextCtx.Act.Log = ctx.Act.Log

		utils.LogDebug(fmt.Sprintf("CmdExec : sub act : start execution [act=%s]", ctx.Act.Name, nextCtx.Args))
		nextCtx.Exec()
		utils.LogDebug(fmt.Sprintf("CmdExec : sub act : end [act=%s]", ctx.Act.Name))

		/**
		 * Now that we finished running the command we need to
		 * release the wait group (i.e., mark it as done).
		 */
		if wg != nil {
			wg.Done()
		}

		return
	}

	/**
	 * Set the command to run (script or command line).
	 */
	var shArgs []string
	var cmdLine string

	if cmd.Script != "" {
		cmdLine = utils.CompileTemplate(cmd.Script, vars)

		var cmdArgs []string

		for _, arg := range cmd.Args {
			compiledArg := utils.CompileTemplate(arg, vars)
			cmdArgs = append(cmdArgs, compiledArg)
		}

		shArgs = append([]string{cmdLine}, cmdArgs...)
	} else {
		cmdLine = utils.CompileTemplate(cmd.Cmd, vars)

		shArgs = []string{"-c", cmdLine, "--"}
	}

	// Set shell to use in the right precedence order.
	shell := "bash"

	if ctx.ActFile.Shell != "" {
		shell = ctx.ActFile.Shell
	}

	if ctx.Act.Shell != "" {
		shell = ctx.Act.Shell
	}

	if cmd.Shell != "" {
		shell = cmd.Shell
	}

	utils.LogDebug(fmt.Sprintf("CmdExec : starting execution [act=%s]", ctx.Act.Name), shArgs)

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
	 * Set a special ACT_ENV_FILE variable pointing to the full
	 * path to env file set on actfile.
	 */
	if ctx.ActFile.EnvFilePath != "" {
		envFilePath := utils.ResolvePath(path.Dir(ctx.ActFile.LocationPath), ctx.ActFile.EnvFilePath)

		vars["ACT_ENV_FILE"] = envFilePath
	}

	/**
	 * Set environment variables using all available variables.
	 */
	envars := ctx.VarsToEnvVars(vars)

	// Set all env vars to shell command.
	shCmd.Env = envars

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
	 *
	 * @NOTE : For some reason using SysProcAttr.Setpgid give us some
	 * weird behaviors at least in MacOS. Using SysProcAttr.Setsid seems
	 * to have the same end result (creating different pgid for child
	 * process). Based on the following:
	 *
	 * https://stackoverflow.com/questions/43364958/start-command-with-new-process-group-id-golang
	 */
	shCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	/**
	 * Set output
	 */
	if !ctx.RunCtx.Quiet && !ctx.Act.Quiet && !ctx.CurrentStage.Quiet && !cmd.Quiet {

		/**
		 * Set the log mode. By default log mode is `raw` and therefore we going
		 * to send all logs directly to stdout without any prefixing containing
		 * act info. If we want to prepend log lines with a prefix containing
		 * act name id and timestamp we can set log mode as `prefixed`.
		 */
		logMode := getLogMode(cmd, ctx)

		if !ctx.RunCtx.IsDaemon && logMode == "raw" {
			shCmd.Stdout = os.Stdout
			shCmd.Stderr = os.Stderr
			shCmd.Stdin = os.Stdin
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

	// Save to run context info file
	ctx.RunCtx.Info.AddCmdPgid(pgid)

	/**
	 * Wait command finalization and get any error code thrown.
	 *
	 * @note: When we kill the main process we going to run KillChildren
	 * function to kill all children. In this case shCmd.Wait going
	 * to rise an error because the command got killed.
	 */
	if err := shCmd.Wait(); err != nil && !ctx.RunCtx.IsFinishing {
		if exiterr, ok := err.(*exec.ExitError); ok {
			errMsg := fmt.Sprintf("command '%s' failed", cmdLine)

			/**
			 * Program exited with exit code other then 0 (which means
			 * an error happened). This works both on Unix and Windows.
			 *
			 * Code got from:
			 *
			 * https://stackoverflow.com/questions/10385551/get-exit-code-go
			 */
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitStatus := status.ExitStatus()

				if exitStatus > 0 {
					/**
					 * We don't want to exit from main process when we are
					 * running commands in parallel but we want to get
					 * notified about command failure.
					 */
					if ctx.CurrentStage.Parallel {
						utils.LogError(errMsg, err)
					} else {
						utils.FatalErrorWithCode(status.ExitStatus(), errMsg, err)
					}
				}
			} else {
				if ctx.CurrentStage.Parallel {
					utils.LogError(errMsg, err)
				} else {
					utils.FatalError(errMsg, err)
				}
			}
		}
	}

	utils.LogDebug(fmt.Sprintf("CmdExec : wait done [act=%s]", ctx.Act.Name), shArgs)

	/**
	 * Now that the command finished let's remove its pgid.
	 */
	ctx.RunCtx.Info.RmCmdPgid(pgid)

	/**
	 * Now that we finished running the command we need to
	 * release the wait group (i.e., mark it as done).
	 */
	if wg != nil {
	 wg.Done()
	}
}
