package run

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/logrusorgru/aurora/v3"
	"github.com/nosebit/act/actfile"
	"github.com/nosebit/act/utils"
	"github.com/teris-io/shortid"
)

//############################################################
// Types
//############################################################
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
	 * Run context info as stored in act data dir.
	 */
	Info *Info

	/**
	 * Flag indicating we are killing the run process.
	 */
	IsKilling bool

	/**
	 * Flag indicating if we are running the process as a
	 * daemon in the background.
	 */
	IsDaemon bool

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
func CreateRunCtx(args []string, actFile *actfile.ActFile) *RunCtx {
	nameId := args[0]
	actNames := strings.Split(nameId, ActCallIdSeparator)

	// Create run context to be filled
	ctx := &RunCtx{
		ActFile:     actFile,
		Vars:        make(map[string]string),
		EnvFileVars: make(map[string]string),
		ActVars:     make(map[string]string),
		Args:        args[1:],
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

	ctx.ActCtx = actCtx
	ctx.ActCtx.Args = ctx.Args

	return ctx
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
	runCtx = CreateRunCtx(cmdArgs, actFile)

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
	} else {
		/**
		 * We save info file just when we are running in not daemon mode because when we
		 * run in daemon mode the only thing act going to do is to spawn another act run
		 * command in the background (not daemon).
		 */
		runCtx.Info.Save()

		// Now run the matched act
		runCtx.ActCtx.Exec()

		// Now that we are done lets clean
		runCtx.Info.RmDataDir()
	}
}

/**
 * This function going to cleanup everything for this command on exit.
 */
func Cleanup() {
	/**
	 * If we have a running act let's kill it and all it's descendant
	 * children (as part of killing the process group as a whole).
	 */
	if runCtx != nil {
		fmt.Println("")
		runCtx.Info.Kill()
	}

	// Exit main process
	os.Exit(0)
}
