package run

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
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
	 * Run context info as stored in act data dir.
	 */
	Info *Info

	/**
	 * Flag indicating we are killing the run process.
	 */
	IsKilling bool

	/**
	 * Flag indicating we should supress all logs.
	 */
	Quiet bool
}

//############################################################
// RunCtx Struct Functions
//############################################################
func (ctx *RunCtx) Print() {
	ctx.ActCtx.Print()
}

/**
 * This function going to kill the run context.
 */
func (ctx *RunCtx) Kill() {
	ctx.IsKilling = true

	for _, pgid := range ctx.Info.Pgids {
		// Kill the whole process group
		err := syscall.Kill(-pgid, syscall.SIGKILL)

		if err != nil {
			utils.FatalError(fmt.Sprintf("could not kill process pgid=%d", pgid), err)
		}
	}

	ctx.Info.RmDataDir()
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
		ActFile:          actFile,
		Vars:             make(map[string]string),
		Args:             args[1:],
	}

	// Create run info
	var execId string

	if id, present := os.LookupEnv("ACT_DAEMON_ID"); present {
		execId = id
	} else {
		id, _ := shortid.Generate()
		execId = id
	}

	ctx.Info = &Info{
		Id:   execId,
		NameId: nameId,
	}

	// Load vars from env file
	if ctx.ActFile.EnvFilePath != "" {
		envFilePath := utils.ResolvePath(path.Dir(ctx.ActFile.LocationPath), ctx.ActFile.EnvFilePath)

		envars, _ := godotenv.Read(envFilePath)

		for key, val := range envars {
			ctx.Vars[key] = val
		}

		/**
		 * Load variables to os env as well so they are passed
		 * over to all commands.
		 */
		godotenv.Load(envFilePath)
	}

	// Set run context variables
	ctx.Vars["ActEnv"] = ctx.Info.GetEnvVarsFilePath()

	// Find the act context to run
	ctx.ActCtx = FindActCtx(actNames, actFile, nil, ctx)
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

	// To run this act in daemon we going to spawn act run.
	if *daemonPtr {
		cmdLineArgs := []string{"run", fmt.Sprintf("-f=%s", actFilePath), runCtx.Info.NameId}
		cmdLineArgs = append(cmdLineArgs, runCtx.Args...)

		shCmd := exec.Command("act", cmdLineArgs...)
		shCmd.Dir = utils.GetWd()
		shCmd.Env = append(os.Environ(), fmt.Sprintf("ACT_DAEMON_ID=%s", runCtx.Info.Id))
		shCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		/**
		 * Set output to file
		 */
		os.MkdirAll(runCtx.Info.GetDataDirPath(), 0755)

		logFile, err := os.OpenFile(runCtx.Info.GetLogFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			utils.FatalError("could not open log file", err)
		}

		shCmd.Stdout = logFile
		shCmd.Stderr = logFile

		// Start
		if err := shCmd.Start(); err != nil {
			utils.FatalError("could not start", err)
		}

		fmt.Printf("😎 started with id %s\n", aurora.Green(runCtx.Info.Id).Bold())
	} else {
		// Save run info
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
		runCtx.Kill()
	}

	// Exit main process
	os.Exit(0)
}
