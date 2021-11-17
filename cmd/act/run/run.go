package run

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/logrusorgru/aurora/v3"
	"github.com/nosebit/act/cmd/act/actfile"
	"github.com/nosebit/act/cmd/act/utils"
	"github.com/teris-io/shortid"
)

//############################################################
// Types
//############################################################
/**
 * Execution state.
 */
const (
	ExecStateStopped string = "stopped"
	ExecStateRunning = "running"
)

/**
 * This run context going to hold all global info we need to run
 * an act.
 */
type RunCtx struct {
	/**
	 * Cli arguments passed by the user.
	 */
	Args []string

	/**
	 * This is the act ctx we going to execute.
	 */
	ActCtx *ActRunCtx

	/**
	 * This is the root actfile.
	 */
	ActFile *actfile.ActFile

	/**
	 * This is a set indicating which actfiles were already loaded.
	 */
	LoadedActFiles map[string]bool

	/**
	 * This are global variables to be used by all acts in the stack.
	 */
	Vars map[string]string

	/**
	 * Set of variables loaded from file.
	 */
	EnvFileVars map[string]string

	/**
	 * Act runtime variables.
	 */
	ActVars map[string]string

	/**
	 * This stack going to hold all acts run contexts we have
	 * active so far.
	 */
	ActCtxCallStack []*ActRunCtx

	/**
	 * Run context info as stored in act data dir.
	 */
	Info *Info

	/**
	 * Flag indicating if we are running the process as a
	 * daemon in the background.
	 */
	IsDaemon bool

	/**
	 * Flag indicating the state of the execution.
	 */
	State string

	/**
	 * Flag indicating we are finishing the execution.
	 */
	IsFinishing bool

	/**
	 * Log mode.
	 */
	Log string

	/**
	 * Flag indicating we should supress all logs.
	 */
	Quiet bool
}

//############################################################
// RunCtx Struct Functions
//############################################################

/**
 * This function going to print all info about this run context.
 */
func (ctx *RunCtx) Print() {
	ctx.ActCtx.Print()
}

//############################################################
// Internal Variables
//############################################################
var runCtx *RunCtx

//############################################################
// Internal Functions
//############################################################
/**
 * This function creates a new run context.
 */
func createRunCtx(args []string, actFile *actfile.ActFile) *RunCtx {
	nameId := args[0]
	actNames := strings.Split(nameId, ActCallIdSeparator)

	// Create run context to be filled
	ctx := &RunCtx{
		ActFile:     	actFile,
		Vars:        	make(map[string]string),
		EnvFileVars: 	make(map[string]string),
		ActVars:     	make(map[string]string),
		Args:        	args[1:],
	}

	// Create run info
	var runId string

	if id, present := os.LookupEnv("ACT_RUN_ID"); present {
		os.Unsetenv("ACT_RUN_ID")
		runId = id
	} else {
		id, _ := shortid.Generate()
		runId = id
	}

	ctx.Info = &Info{
		Id:     runId,
		NameId: nameId,
	}

	/**
	 * If parent process invoked this process as a daemon
	 * then lets flag it. This going to have impact on how
	 * we do logging fot the commands in cmd.go file.
	 */
	if _, present := os.LookupEnv("ACT_DAEMON"); present {
		os.Unsetenv("ACT_DAEMON")
		ctx.IsDaemon = true
	}

	/**
	 * If this act processes was invoked by another parent act
	 * process then we going to adjust the act name id to include
	 * parent name id. This way if the parent process is called
	 * foo and this child process is called bar then the name id
	 * we going to use is foo::bar.
	 */
	if parentId, present := os.LookupEnv("ACT_PARENT_RUN_ID"); present {
		os.Unsetenv("ACT_PARENT_RUN_ID")

		ctx.Info.ParentActId = parentId

		parentInfo := GetInfo(parentId)

		if parentInfo == nil {
			utils.FatalError("parent process not found")
		}

		ctx.Info.NameId = fmt.Sprintf("%s::%s", parentInfo.NameId, ctx.Info.NameId)
	}

	// Get process group id
	pid := os.Getpid()
	pgid, err := syscall.Getpgid(pid)

	if err != nil {
		utils.FatalError("could not get main process groupd id", err)
	}

	ctx.Info.Pid = pid
	ctx.Info.Pgid = pgid

	// Set run context variables
	ctx.ActVars["ActEnv"] = ctx.Info.GetEnvVarsFilePath()

	// Find the act context to run
	actCtx, err := FindActCtx(actNames, actFile, nil, ctx)

	if err != nil {
		utils.FatalError(err)
	}

	if actCtx != nil {
		ctx.ActCtx = actCtx
		ctx.ActCtx.Args = ctx.Args
	}

	return ctx
}

func cleanup() {
	utils.LogDebug("cleanup")

	if runCtx != nil && runCtx.ActCtx != nil {
		stack := runCtx.ActCtxCallStack

		utils.LogDebug("cleanup : stack size", len(stack))

		/**
		 * The last context in the stack is the active one so we start
		 * from it and go back through active contexts.
		 */
		for i := len(stack)-1; i >= 0; i-- {
			ctx := stack[i]
			utils.LogDebug("cleanup : running final steps", ctx.Act.Name)
			ctx.FinalStageExec()
	 	}
	}

	// Now that we are done lets clean
	runCtx.Info.RmDataDir()
}

//############################################################
// Exported Functions
//############################################################
/**
 * This function to execute run command.
 */
func Exec(args []string) {
	// Set default actfile path.
	defaultActFilePath := "actfile.yml"

	/**
	 * We create a new flag set to allow this act subcommand to
	 * accepts flags by their own.
	 */
	cmdFlags := flag.NewFlagSet("run", flag.ExitOnError)

	/**
	 * This flag indicates if we should run the act as a daemon
	 * in the background instead of running it as a regular
	 * process in the foreground.
	 */
	daemonPtr := cmdFlags.Bool("d", false, "Run act as a daemon in the background")

	/**
	 * This flag allow user to supress all logs.
	 */
	quietPtr := cmdFlags.Bool("q", false, "Supress all logs")

	/**
	 * This flag force raw output.
	 */
	logPtr := cmdFlags.String("l", "", "Log mode")

	/**
	 * This is the path to actfile to be used.
	 */
	actFilePathPtr := cmdFlags.String("f", defaultActFilePath, "Path to an actfile yaml file")

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

	// We read/parse actfile.yml file from current working dir
	wdir := utils.GetWd()
	actFilePath := utils.ResolvePath(wdir, *actFilePathPtr)
	actFile := actfile.ReadActFile(actFilePath)

	// Build run context
	runCtx = createRunCtx(cmdArgs, actFile)

	// Set state as running
	runCtx.State = ExecStateRunning

	// Set quiet logs from command line
	runCtx.Quiet = *quietPtr

	// Set raw logging mode
	runCtx.Log = *logPtr

	// To run this act in daemon we going to spawn act run.
	if *daemonPtr {
		cmdLineArgs := []string{"run", fmt.Sprintf("-f=%s", actFilePath), runCtx.Info.NameId}
		cmdLineArgs = append(cmdLineArgs, runCtx.Args...)

		/**
		 * Set environment variables that going to control
		 * spawned daemon process.
		 */
		envars := []string{
			fmt.Sprintf("ACT_RUN_ID=%s", runCtx.Info.Id),
			"ACT_DAEMON=true",
		}

		shCmd := exec.Command("act", cmdLineArgs...)
		shCmd.Dir = utils.GetWd()
		shCmd.Env = append(os.Environ(), envars...)

		// Ensure we create a new session for the new pocess (this means a new pgid)
		shCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		/**
		 * Daemon processes going to log directly to a log file
		 * instead of to stdout.
		 */
		os.MkdirAll(runCtx.Info.GetDataDirPath(), 0755)

		logFile, err := os.OpenFile(runCtx.Info.GetLogFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			utils.FatalError("could not open log file", err)
		}

		shCmd.Stdout = logFile
		shCmd.Stderr = logFile

		/**
		 * Start the process and don´t wait it since its a daemon.
		 */
		if err := shCmd.Start(); err != nil {
			utils.FatalError("could not start", err)
		}

		fmt.Printf("😎 started with id %s\n", aurora.Green(runCtx.Info.Id).Bold())
	} else if runCtx.ActCtx != nil {
		/**
		 * We save info file just when we are running in not daemon mode because when we
		 * run in daemon mode the only thing act going to do is to spawn another act run
		 * command in the background (not daemon).
		 */
		runCtx.Info.Save()

		// Now run the matched act
		runCtx.ActCtx.Exec()

		utils.LogDebug("Exec : done")

		/**
		 * Let's run final commands only when exec finished naturally
		 * (i.e., not killed). In this scenario the execution going to
		 * be still running.
		 */
		/*if runCtx.State != ExecStateStopped {
			utils.LogDebug("Exec : cleanup call")
			cleanup()
		}*/
	}
}

/**
 * This function going to stop execution of current running
 * commands.
 */
func Stop() {
	utils.LogDebug(fmt.Sprintf("Stop [State=%s]", runCtx.State))

	/**
	 * Stop only if we are executing non final commands.
	 */
	if runCtx != nil && !runCtx.IsFinishing && runCtx.State == ExecStateRunning {
		/**
		 * If we have a running act let's kill it and all it's descendant
		 * children (as part of killing the process group as a whole).
		 */
		if runCtx.ActCtx != nil {
			// First we kill current running context.
			runCtx.Info.KillChildren();
		}

		runCtx.State = ExecStateStopped
	}
}

/**
 * This function going to cleanup everything for this command on exit.
 */
func Finish() {
	utils.LogDebug(fmt.Sprintf("Finish [State=%d]", runCtx.State), runCtx.IsFinishing)

	/**
	 * In case user tries to kill this process twice we going to
	 * prevent running final actions multiple times.
	 */
	if runCtx == nil || runCtx.IsFinishing {
		return
	}

	/**
	 * If we called Finish at the end of main process (i.e. in main.go)
	 * then everything went fine and user didn't kill the process.
	 * This way we can skip this finish process because the final step
	 * was already done in act run ctx exec function.
	 */
	if runCtx.State == ExecStateRunning {
		/**
		 * We call KillChildren because we might have some dangling
		 * detached child acts running and we want to kill them.
		 */
		runCtx.Info.KillChildren();
		runCtx.Info.RmDataDir()
		return
	}

	/**
	 * Since we are reusing the same run context to run finishing
	 * commands we need to resume the run context first to allow
	 * new execution.
	 */
	runCtx.State = ExecStateRunning

	/**
	 * Set the flag isFinishing to run context so we can propagate
	 * this information down to the process tree.
	 */
	runCtx.IsFinishing = true

	/**
	 * If we have a running act let's kill it and all it's descendant
	 * children (as part of killing the process group as a whole).
	 */
	if runCtx.ActCtx != nil {
		utils.LogDebug("Finish : cleanup call")

		// If act has teardown commands let's run them before exit.
		cleanup()
	}
}
