/**
 * This file implements the run subcommand which is responsible
 * for executing acts in both background (daemon) and foreground
 * modes.
 */

package cmd

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"syscall"

	"github.com/nosebit/act/actfile"
	"github.com/nosebit/act/utils"
)

//############################################################
// Types
//############################################################
/**
 * This struct going to hold everything required for the correct
 * execution of a specific act.
 */
type ActRunCtx struct {
	/**
	 * This is the actfile the actually hold the act we are executing.
	 * In the case of a subact injected with `from` or `include`
	 * statements this actfile going to be the one holding the subact
	 * definition (actfile specified in from or include statements). So
	 * in the case we invoke `act run act1 ... actN` then TailActFile
	 * is the actfile which holds the definition of actN.
	 */
	TailActFile *actfile.ActFile

	/**
	 * This is the actfile the originated the whole subcommand chain,
	 * i.e., the actfile the holds the definition of the first act
	 * we find in invokation `act run act1 ... actN`. To be more
	 * precise, HeadActFile refers to act1 and TailActFile refers to
	 * actN.
	 */
	HeadActFile *actfile.ActFile

	/**
	 * This uniquely identifies the act we are running and it's
	 * an MD5 hash of act call id inside the tail actfile (i.e.,
	 * the actfile we actually found the act to run) together with
	 * the actfile location path. So supose we have:
	 *
	 * ```yaml
	 * # /path/to/actfile.yml
	 * acts:
	 *   foo:
	 *     acts:
	 *       bar:
	 *         cmds:
	 *           - echo "im bar"
	 * ```
	 * Then `bar` act id going to be a MD5 hash of the following
	 * string: `/path/to/actfile.yml:foo.bar`. This is going to be
	 * used in commands that can call other acts because we going
	 * to need a way to uniquely create a shell function call
	 * that represents the act. So in the case we have:
	 *
	 * ```yaml
	 * # /path/to/actfile.yml
	 * acts:
	 *   zoo:
	 *     cmds:
	 *       - echo "starting zoo"
	 *       - act: foo.bar
	 *       - echo "finishing zoo"
	 *   foo:
	 *     acts:
	 *       bar:
	 *         cmds:
	 *           - echo "im bar"
	 * ```
	 *
	 * the second command o zoo calls the act `foo.bar`. In this
	 * case when we run `act run zoo` the final script we going
	 * to execute going to look something like this:
	 *
	 * ```bash
	 * # Act zoo in /path/to/actfile.yml
	 * 3f143bb8c6edd78a54d1b0f42211e208() {
	 *	 echo "starting zoo"
	 *   4098f3ba2ce1c48f8d7849fb173630bd
	 *   echo "finishing zoo"
	 * }
	 *
	 * # Act foo.bar in /path/to/actfile.yml
	 * 4098f3ba2ce1c48f8d7849fb173630bd() {
	 * 	 echo "im bar"
	 * }
	 *
	 * # Execute zoo
	 * 3f143bb8c6edd78a54d1b0f42211e208
	 * ```
	 */
	ActId string

	/**
	 * This is the full subact chain name we used when calling the
	 * act in the run command. When we call `act run foo.bar` for
	 * example to execute the bar subact of foo act then the ActCallId
	 * is `foo.bar` for the bar act in this case. This uniquely
	 * identifies the user call for executing bar act but keep in
	 * mind that we can reach the same `bar` act in different
	 * call ways. For example, if we have the following:
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   zoo:
	 *     include: another/actfile.yml
	 * ```
	 * ```
	 * # another/actfile.yml
	 * acts:
	 *   foo:
	 *     acts:
	 *       bar:
	 *         cmds:
	 *           - echo "im bar"
	 * ```
	 *
	 * then we can call the exactly same `bar` act with
	 * `act run zoo.foo.bar`.
	 */
	ActCallId string

	/**
	 * This is the name of executing act and correspond to the last
	 * command name of ActNameId, therefore this does no consider
	 * subcommands path as ActNameId do. So following the example in
	 * the comment for ActNameId if we invoke `act run foo bar` then
	 * ActName going to be just `bar`.
	 */
	ActName string

	/**
	 * Variables that can be used by the act. We can use variables in
	 * too different ways:
	 *
	 * (1) In act compile time using go template syntax like the
	 *     following:
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "name is {{ .actName }}"
	 * ```
	 *
	 * In this case we going to compile the command to be executed
	 * before execution replacing actName with the actual act name
	 * value. So when it's time to execute this act command we going
	 * to simply execute `echo "name is foo"` shell command.
	 *
	 * (2) In act execution time using environment variables like
	 *     the following:
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "name is $ACT_NAME"
	 * ```
	 *
	 * In this case the env variable is used diretly during the execution
	 * of the command.
	 *
	 * The good benefit of (1) is that we can use some super powers of
	 * go template language directly (like looks, etc) but we cannot
	 * use variables like that when the act is executed through a script
	 * for example using the script field or placing script file in
	 * `acts/foo.sh` location. When running act through script the only
	 * way is to use env variables as in (2). Unless someone wants to
	 * implement compilation of the whole script using go tamplate
	 * language (@TODO).
	 */
	Vars map[string]interface{}

	/**
	 * This is the file system path to a directory which going to hold
	 * all info about the running act. This includes info like the act
	 * process id (pid), logs, etc. For an act named `foo` defined in
	 * /path/to/actfile.yml this data directory going to be located in
	 * /path/to/.actdt/foo. Similarly for subacts like `foo bar` we going
	 * to have data folder in /path/to/.actdt/foo/bar.
	 */
	ActDataDirPath string

	/**
	 * This is the act we are executing.
	 */
	Act *actfile.Act

	SubActNameChain []string
	SubActs         []*actfile.Act

	/**
	 * This is the process id associated with act execution.
	 */
	Pid int

	/**
	 * This is the process group id associated with act execution.
	 * This is used to kill all child process the act might have
	 * spawned.
	 */
	Pgid int

	/**
	 * After removing all subacts and flags from original args we
	 * end up with this RestArgs which is arguments we should pass
	 * over to the act we going to execute.
	 */
	RestArgs []string
}

//############################################################
// Internal Variables
//############################################################
var runCtx *ActRunCtx

//############################################################
// Internal Functions
//############################################################

/**
 * This function going to recursivelly traverse actfiles to
 * best match an act to be executed.
 *
 * @param actNames - List of subacts name chain.
 * @param ctx - The context info collected so far.
 */
func fillActRunCtxRec(actNames []string, ctx *ActRunCtx) {
	/**
	 * The first argument is supposed to be the next act name we
	 * want to match. So suppose we run `act run foo bar`. After
	 * parsing flags for `run` subcommand we going to end up with
	 * `foo bar` arguments. In the first round we are targeting
	 * to match an act named `foo` which is args[0].
	 */
	targetActName := actNames[0]

	utils.LogDebug(">>> MATCHING NAME", targetActName, ctx.TailActFile.LocationPath)

	/**
	 * Store current actfile in case we does not match in any
	 * other included actfile.
	 */
	currentActFile := ctx.TailActFile

	/**
	 * Acts to match.
	 */
	var actsToMatch []*actfile.Act

	/**
	 * If we already macthed an act that has subacts then lets
	 * use those subacts so we can continue matching.
	 */
	if ctx.SubActs != nil {
		actsToMatch = ctx.SubActs
	} else {
		actsToMatch = currentActFile.Acts
	}

	/**
	 * Iterate over all defined act names in current actfile and
	 * pick up the best match.
	 */
	for _, act := range actsToMatch {
		/**
		 * The act name is actually a regex which we are going to use
		 * to match against user provided act name. This is very
		 * useful becase we can have actfiles like this:
		 *
		 * ```yaml
		 * # actfile.yml
		 * acts:
		 *   foo-.+:
		 *     cmds:
		 *       - echo "im $ACT_NAME"
		 * ```
		 *
		 * which going to match when running `act run foo-bar` for
		 * example.
		 */
		match, _ := regexp.MatchString(fmt.Sprintf("^%s$", act.Name), targetActName)

		/**
		 * If actName does not match simply continue to next
		 * defined act name in the actfile.
		 */
		if !match {
			continue
		}

		utils.LogDebug(fmt.Sprintf("act %s matched with %s in %s", targetActName, act.Name, currentActFile.LocationPath))

		// Set ctx act name and id
		ctx.ActName = targetActName

		// Set vars for rendering text templates
		ctx.Vars["actName"] = ctx.ActName

		/**
		 * If we matched an act which contains a `from` field defined
		 * then this means we want to forward the execution to
		 * another actfile which contains an act with the same name.
		 * So, for example, if we have the following actfiles:
		 *
		 * ```yaml
		 * # actfile.yml
		 * acts:
		 *   foo:
		 *     from: another/actfile.yml
		 * ```
		 *
		 * and
		 *
		 * ```yaml
		 * # another/actfile.yml
		 * acts:
		 *   foo:
		 *     cmds:
		 *       - echo "im foo in another actfile"
		 * ```
		 *
		 * then when running `act run foo` in the folder containing
		 * actfile.yml file we going to see "im foo in another actfile"
		 * printed to the screen.
		 */
		if act.From != "" {
			from := utils.CompileTemplate(act.From, ctx.Vars)

			// Set next actfile to inspect.
			ctx.TailActFile = actfile.ReadActFile(utils.ResolvePathFromWd(from))

			/**
			 * When using from statement we are kind of linking two acts with
			 * same name in different actfiles. So an act `foo` in `actfile.yml`
			 * which points to another `foo` act in `another/actfile.yml`
			 * going to have a call id of only `foo` instead of `foo.foo`.
			 */
			ctx.ActCallId = path.Dir(ctx.ActCallId)

			/**
			 * Set to nil because we are changing actfile and therefore we
			 * don't want to pass a macthed subact.
			 */
			ctx.SubActs = nil
			ctx.SubActNameChain = []string{}

			// Keep filling in another actfile.
			fillActRunCtxRec(actNames, ctx)
		}

		/**
		 * If we matched an act that contains an `include` field defined
		 * then we going to place subacts under matched act coming from
		 * a different actfile. So lets say we have the follwing:
		 *
		 * ```yaml
		 * # actfile.yml
		 * acts:
		 *   foo:
		 *     include: another/actfile.yml
		 * ```
		 *
		 * and
		 *
		 * ```yaml
		 * # another/actfile.yml
		 * acts:
		 *   bar:
		 *     cmds:
		 *       - echo "im bar in another actfile"
		 * ```
		 *
		 * then user can run `act run foo bar` to see "im bar in another
		 * actfile" poping in screen.
		 */
		if act.Include != "" && len(actNames) > 0 {
			utils.LogDebug("processing include", act.Include)

			include := utils.CompileTemplate(act.Include, ctx.Vars)

			// Set next actfile to inspect.
			ctx.TailActFile = actfile.ReadActFile(utils.ResolvePathFromWd(include))

			/**
			 * Set to nil because we are changing actfile and therefore we
			 * don't want to pass a macthed subact.
			 */
			ctx.SubActs = nil
			ctx.SubActNameChain = []string{}

			// Keep filling in another actfile now.
			fillActRunCtxRec(actNames[1:], ctx)
		}

		/**
		 * If act has subacts then lets try to keep matching inside the same
		 * actfile.
		 */
		if len(act.Acts) > 0 && len(actNames) > 0 {
			ctx.SubActs = act.Acts
			ctx.SubActNameChain = append(ctx.SubActNameChain, targetActName)

			utils.LogDebug("act has subacts", act.Name, ctx.ActId, ctx.TailActFile.LocationPath)

			fillActRunCtxRec(actNames[1:], ctx)
		} else if len(ctx.SubActNameChain) > 0 {
			/**
			 * In this case we know that this is a subact of a previously
			 * defined act.
			 */
			ctx.SubActNameChain = append(ctx.SubActNameChain, targetActName)
		}

		/**
		 * When traversing to from/include it could happen we
		 * found a subact to run. If not let's set this one as
		 * the one to run.
		 */
		if ctx.Act == nil {
			ctx.Act = act
			ctx.TailActFile = currentActFile
		}

		/**
		 * Otherwise we going to get the macthed act as the correct act
		 * (bc we could not match any subact).
		 */
		//ctx.ActId = targetActName

		/**
		 * We need to hash the act name relative to actfile location to generate
		 * the id.
		 */

		// Finish processing
		return
	}

	/**
	 * If we land up here then we were not able to find an act
	 */
	utils.FatalError("act not found", targetActName)
}

//############################################################
// Exposed Functions
//############################################################

/**
 * This function going to find the correct exec context for the
 * act we want to execute.
 *
 * @param actCallId - Call id used in act run command.
 * @param actFile - The main actfile from which user invoked the
 *   `act run` command.
 */
func FindActRunCtx(actCallId string, actFile *actfile.ActFile) *ActRunCtx {
	/**
	 * Create an empty exec context that going to be populated.
	 */
	runCtx := &ActRunCtx{
		ActCallId:   actCallId,
		HeadActFile: actFile,
		TailActFile: actFile,
		Vars:        map[string]interface{}{},
	}

	// Set some vars already
	runCtx.Vars["actCallId"] = runCtx.ActCallId
	runCtx.Vars["actPathId"] = strings.Split(runCtx.ActCallId, utils.ActCallIdSeparator)

	// We get the subact names chain as a list from actNameId.
	actNames := strings.Split(actCallId, utils.ActCallIdSeparator)

	// Now we fill run context recursively.
	fillActRunCtxRec(actNames, runCtx)

	if runCtx.Act == nil {
		utils.FatalError("could not find an act to execute")
	}

	/**
	 * We create an act id hash from subacts chain call done inside
	 * the same actfile.
	 */
	var subActNamesChain string

	if len(runCtx.SubActNameChain) > 0 {
		subActNamesChain = strings.Join(runCtx.SubActNameChain, utils.ActCallIdSeparator)
	} else {
		subActNamesChain = runCtx.ActName
	}

	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", runCtx.TailActFile.LocationPath, subActNamesChain)))

	runCtx.ActId = hex.EncodeToString(hash[:])
	runCtx.Act.Id = runCtx.ActId
	runCtx.Act.CallId = actCallId
	runCtx.ActDataDirPath = utils.GetActDataDirPath(runCtx.ActCallId)

	return runCtx
}

/**
 * This function going to collect act commands.
 */
func BuildCmdLinesMap(act *actfile.Act, runCtx *ActRunCtx, cmdLinesMap map[string][]string) {
	var cmdLines []string

	for _, cmd := range act.Cmds {
		var cmdLine string

		if cmd.Cmd != "" {
			cmdLine = utils.CompileTemplate(cmd.Cmd, runCtx.Vars)
		} else if cmd.Act != "" {
			newRunCtx := FindActRunCtx(cmd.Act, runCtx.TailActFile)

			// If we found an act we going to process it
			if newRunCtx.Act == nil {
				utils.FatalError(fmt.Sprintf("could not find act %s", cmd.Act))
			}

			cmdLine = fmt.Sprintf("act-%s", newRunCtx.ActId)

			if _, ok := cmdLinesMap[newRunCtx.ActId]; !ok {
				BuildCmdLinesMap(newRunCtx.Act, newRunCtx, cmdLinesMap)
			}
		}

		cmdLines = append(cmdLines, cmdLine)
	}

	cmdLinesMap[act.Id] = cmdLines
}

/**
 * This is the main execution point for the `run` command.
 */
func RunCmdExec(args []string, actFile *actfile.ActFile) {
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
	 * Parse the incoming args extracting defined flags if user
	 * provided any.
	 */
	cmdFlags.Parse(args)

	/**
	 * This are the command line arguments after extracting
	 * the flags.
	 */
	cmdArgs := cmdFlags.Args()

	// Fail if user did not provided an act name.
	if len(cmdArgs) < 1 {
		utils.FatalError("an act name is required")
	}

	/**
	 * First run command arg need to be an act name id which is
	 * a subacts names chain like in foo.bar (chained by dot
	 * symbol).
	 */
	actNameId := cmdArgs[0]

	/**
	 * All other command arguments passed should be passed directly
	 * to the act.
	 */
	//actArgs := cmdArgs[1:]

	/**
	 * The first thing we need to do in order to execute an act
	 * is to find the correct execution context for it. This
	 * execution going to contain info for the act we going to
	 * run as well as other important information like the actfile
	 * where we found the act to run. This is important because
	 * acts in one actfile can reference acts in another actfile
	 * in a recursive way.
	 */
	runCtx = FindActRunCtx(actNameId, actFile)

	if runCtx.Act == nil {
		utils.FatalError("we could not find an act to execute")
	}

	/**
	 * Now we going to run over all act commands so we can create a
	 * shell script with all of them together. We do this instead
	 * of executing each command at a time because this way we can
	 * handle long running acts more properly since we can just spawn
	 * a shell process that is in charge of running the script.
	 */

	/**
	 * We first start by finding out the base directory where the
	 * actfile containinng the act was placed in. This is important
	 * because if act does not provide any cmds or explicit script
	 * file path then we going to look up an associated script file
	 * in some common places inside baseDir. For example, for an act
	 * called `foo` defined in actfile located in `/path/to/actfile.yml`
	 * which does not specify cmds or script we going to look up for
	 * a script named `foo.sh` located in `/path/to/acts/foo.sh`. If
	 * we find this script then we going to use it when running the act.
	 */
	baseDir := path.Dir(runCtx.TailActFile.LocationPath)

	/**
	 * We going to build up the list of commands we going to run when
	 * executing the act.
	 */
	var actCmds []*actfile.Cmd

	/**
	 * First we handle the case where we going to use a script instead
	 * of commands. As mentioned before if act to be executed does
	 * not define cmds or explicit script file path then we going
	 * to look up for an execution script in some default locations.
	 */
	if runCtx.Act.Script != "" || runCtx.Act.Cmds == nil {
		var scriptRelPath string

		if runCtx.Act.Script != "" {
			scriptRelPath = runCtx.Act.Script
		} else {
			scriptRelPath = fmt.Sprintf("acts/%s.sh", runCtx.ActName)
		}

		scriptPath := path.Join(baseDir, scriptRelPath)

		if !utils.DoFileExists(scriptPath) {
			utils.FatalError(fmt.Sprintf("no cmds or script %s found", scriptPath))
		}

		// We push the command to run the script to act list of commands
		// to be executed.
		actCmds = append(actCmds, &actfile.Cmd{
			Cmd: fmt.Sprintf("sh %s", scriptPath),
		})
	} else if runCtx.Act.Cmds != nil {
		actCmds = runCtx.Act.Cmds
	}

	/**
	 * Just to make sure: if we don't have commands to execute then
	 * exit with error.
	 */
	if len(actCmds) == 0 {
		utils.FatalError("nothing to execute")
	}

	// Create act data directory so we can start writing files there.
	os.MkdirAll(runCtx.ActDataDirPath, 0755)

	/**
	 * As mentioned before we going to put all act commands together in
	 * a script file so we can handle long running acts more
	 * easily.
	 *
	 * To allow commands that call another acts we going to wrap acts
	 * in bash functions in the final screen. Only the time going to
	 * tell if this is a wise decision. The good thing about this is
	 * that all commands run together in the same runtime context allowing
	 * them to share variables for example. But one downside for example
	 * is not letting commands to run in parallel. If in the future we
	 * decide to run each command independently (allowing parallel
	 * execution) i think we should have an `execCmd` function that
	 * receives a command and execute it right away. To support long running
	 * acts in this scenario i think `act run -d foo` should spawn
	 * an `act run foo` process and supervise it (start/stop/restart).
	 */
	cmdLinesMap := make(map[string][]string)
	var scriptLines []string

	BuildCmdLinesMap(runCtx.Act, runCtx, cmdLinesMap)

	// Build act script content.
	for actId, cmdLines := range cmdLinesMap {
		scriptLines = append(scriptLines, fmt.Sprintf("act-%s() {", actId))

		for _, cmdLine := range cmdLines {
			scriptLines = append(scriptLines, fmt.Sprintf("  %s", cmdLine))
		}

		scriptLines = append(scriptLines, "}")
		scriptLines = append(scriptLines, "")
	}

	scriptLines = append(scriptLines, fmt.Sprintf("act-%s", runCtx.ActId))

	/**
	 * Write all command lines to act script file (first removing
	 * old file to start fresh).
	 */
	actScriptContent := strings.Join(scriptLines, "\n")

	actScriptFilePath := utils.GetActScriptFilePath(runCtx.ActCallId)

	os.Remove(actScriptFilePath)
	utils.WriteToFile(actScriptFilePath, actScriptContent)

	// Convert vars to a list of env vars to be used in act execution
	envVars := utils.VarsMapToEnvVars(runCtx.Vars)

	/**
	 * Spawn the command to be executed.
	 *
	 * @TODO : We should let user specify the shell he/she want to
	 * use. This could be done in the act level with a "shell" field
	 * maybe.
	 *
	 * @TODO : I think we should use something like https://github.com/mvdan/sh
	 * so we don't need to rely on local bash installed.
	 */
	shCmd := exec.Command("bash", actScriptFilePath)

	/**
	 * Set environment variables using all available env vars plus
	 * env vars built from local vars.
	 */
	shCmd.Env = append(os.Environ(), envVars...)

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

	/**
	 * Setup process standard and error outputs.
	 */
	if *daemonPtr {
		/**
		 * If we are running the act as a daemon (in the background) then
		 * we going to set shell command to redirect it's outputs to
		 * a file instead of to standard output (screen).
		 */
		logFile, err := os.OpenFile(utils.GetActLogFilePath(runCtx.ActCallId), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			utils.FatalError("could not open log file", err)
		}

		/**
		 * @TODO: Is there a way to log error to a different file but still
		 * keeps all logs in a single file so we can prevent everything
		 * chronologically to user? To have errors in different file is
		 * easier to debug.
		 */
		shCmd.Stdout = logFile
		shCmd.Stderr = logFile
	} else {
		/**
		 * Otherwise we set shell command to output to standard output
		 * (screen).
		 */
		shCmd.Stdout = os.Stdout
		shCmd.Stderr = os.Stderr
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

	/**
	 * Set pid and pgid to exec context so we can kill process later
	 * (for example when we kill the main act run command). Check the
	 * init function in main.go file.
	 */
	runCtx.Pid = pid
	runCtx.Pgid = pgid

	// Lets save act info to file
	info := ActRunInfo{
		ActId:     runCtx.ActId,
		ActName:   runCtx.ActName,
		ActCallId: runCtx.ActCallId,
		Pid:       runCtx.Pid,
		Pgid:      runCtx.Pgid,
	}

	SaveActRunInfo(&info)

	/**
	 * If we are running the act in foreground then we going to block
	 * main process while the act is running (using Wait function).
	 */
	if !*daemonPtr {
		shCmd.Wait()

		// When it's done lets cleanup
		utils.RmActDataDir(runCtx.ActCallId)
	}
}

/**
 * This function going to cleanup everything for this command on exit.
 */
func RunCleanup() {
	/**
	 * If we have a running act let's kill it and all it's descendant
	 * children (as part of killing the process group as a whole).
	 */
	if runCtx != nil {
		// Kill the whole process group
		err := syscall.Kill(-runCtx.Pgid, syscall.SIGKILL)

		if err != nil {
			utils.FatalError(fmt.Sprintf("could not kill process (pid=%d/pgid=%d)", runCtx.Pid, runCtx.Pgid), err)
		}

		// Clean up data dir
		utils.RmActDataDir(runCtx.ActCallId)
	}
}
