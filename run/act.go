package run

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
	"github.com/joho/godotenv"
	"github.com/nosebit/act/actfile"
	"github.com/nosebit/act/utils"
)

//############################################################
// Types
//############################################################

/**
 *  This is the context to run an act.
 */
type ActRunCtx struct {
	/**
	 * Reference to global run context.
	 */
	RunCtx *RunCtx

	/**
	 * Actfile where we found this act.
	 */
	ActFile *actfile.ActFile

	/**
	 * The act to run.
	 */
	Act *actfile.Act

	/**
	 * Prev context in the chain.
	 */
	PrevCtx *ActRunCtx

	/**
	 * This is the call id for the act.
	 */
	CallId string

	/**
	 * This flag indicates if commands should be executed
	 * in parallel or not.
	 */
	ParallelExec bool

	/**
	 * List of process groupd ids created by this act.
	 */
	Pgids []int

	/**
	 * List of cli flag values passed by the user.
	 */
	FlagVals map[string]string

	/**
	 * Cli arguments after extracting flags.
	 */
	Args []string

	/**
	 * Act runtime vars.
	 */
	ActVars map[string]string

	/**
	 * Set of variables scoped to act execution.
	 */
	Vars map[string]string
}

//############################################################
// ActRunCtx Struct Functions
//############################################################

/**
 * This is an utilitary function that going to print the content
 * of this act run context. We get the whole act run context stack
 * so we can print the whole chain from the first one.
 */
func (ctx *ActRunCtx) Print() {
	// Print the whole chain
	stack := ctx.Stack()

	for _, currCtx := range stack {
		fmt.Println("Act", currCtx.CallId)
		fmt.Println("  actFile", currCtx.ActFile.LocationPath)
	}
}

/**
 * This function going to merge all variables altogether.
 */
func (ctx *ActRunCtx) MergeVars() map[string]string {
	vars := make(map[string]string)

	runtimeVars, _ := godotenv.Read(ctx.RunCtx.Info.GetEnvVarsFilePath())

	// Get vars from file
	envFileVars := make(map[string]string)

	if ctx.ActFile.EnvFilePath != "" {
		envFilePath := utils.ResolvePath(path.Dir(ctx.ActFile.LocationPath), ctx.ActFile.EnvFilePath)
		envars, _ := godotenv.Read(envFilePath)
		envFileVars = envars
	}

	environVars := make(map[string]string)

	// Iterate over environ vars
	for _, kv := range os.Environ() {
		parts := strings.Split(kv, "=")

		if len(parts) == 2 {
			environVars[parts[0]] = parts[1]
		}
	}

	varsMapList := []map[string]string{
		// Variables from the enviornment going to be overriden.
		environVars,

		// Load vars from files first.
		envFileVars,

		// Global vars has precedence over vars loaded from file.
		ctx.RunCtx.Vars,

		// Local vars has precedence over global vars.
		ctx.Vars,

		// Runtime vars has precedence over local vars.
		runtimeVars,

		// Act runtime global vars has precedence over all other vars.
		ctx.RunCtx.ActVars,

		// Act runtime local vars has precedence over all other vars.
		ctx.ActVars,

		// Flag vars has precedence over all other vars.
		ctx.FlagVals,
	}

	for _, varsMap := range varsMapList {
		for key, val := range varsMap {
			vars[key] = val
		}
	}

	return vars
}

/**
 * This function convert vars to env vars.
 */
func (ctx *ActRunCtx) VarsToEnvVars(vars map[string]string) []string {
	var envars []string
	actVarNamesMap := make(map[string]bool)

	for key, _ := range ctx.RunCtx.ActVars {
		actVarNamesMap[key] = true;
	}

	for key, _ := range ctx.ActVars {
		actVarNamesMap[key] = true;
	}

	for key, val := range vars {
		theKey := key

		if _, present := actVarNamesMap[key]; present {
			theKey = utils.CamelToSnakeUpperCase(key)
		}

		envars = append(envars, fmt.Sprintf("%s=%s", theKey, val))
	}

	return envars
}

/**
 * This function going to get the whole act run context stack
 * starting from this act run context. Act contexts are linked
 * together (linked list) but it could happen that multiple
 * act run context has the same previous context in this liked
 * list (this happen when we run commands in parallel that call
 * other acts). This way is useful to be able to get the whole
 * stack of act contexts starting from any node in this linked
 * list (for printing for example).
 */
func (ctx *ActRunCtx) Stack() []*ActRunCtx {
	// Print the whole chain
	var stack []*ActRunCtx

	currCtx := ctx

	for currCtx != nil && currCtx.PrevCtx != nil {
		stack = append([]*ActRunCtx{currCtx}, stack...)
		currCtx = currCtx.PrevCtx
	}

	return stack
}

/**
 * This function going to run all before acts not already
 * executed for the whole act run context chain.
 */
func (ctx *ActRunCtx) ExecBeforeAll() {
	var stack []*ActRunCtx
	currCtx := ctx

	/**
	 * Go back in stack until we get the first actfile
	 * which before act was not run yet. We are doing this
	 * way because when running commands in parallel we can
	 * get multiple act ctxs pointing to the same act ctx
	 * as their prev act ctx.
	 */
	for currCtx != nil {
		/**
		 * We assume that all previous before acts were run.
		 */
		if currCtx.ActFile.InitWg != nil {
			break
		}

		currCtx.ActFile.InitWg = &sync.WaitGroup{}

		beforeAll := currCtx.ActFile.BeforeAll

		if beforeAll != nil && len(beforeAll.Cmds) > 0 {
			currCtx.ActFile.InitWg.Add(1)

			beforeCallId := fmt.Sprintf("%s::before", currCtx.CallId)

			beforeAllCtx := ActRunCtx{
				CallId:  beforeCallId,
				ActFile: currCtx.ActFile,
				Act:     beforeAll,
				RunCtx:  runCtx,
				Vars:    runCtx.Vars,
			}

			stack = append([]*ActRunCtx{&beforeAllCtx}, stack...)
		}

		currCtx = ctx.PrevCtx
	}

	// Execute all before acts that were not executed yet.
	for _, currCtx := range stack {
		currCtx.Exec()
	}
}

/**
 * This function going to execute an act.
 */
func (ctx *ActRunCtx) Exec() {
	// First thing we execute all before acts not executed yet.
	ctx.ExecBeforeAll()

	/**
	 * We allow user to specify command line flags for acts. This
	 * way we can have something like this:
	 *
	 * ```yaml
	 * # actfile.yml
	 * version: 1
	 *
	 * acts:
	 *   foo:
	 *     flags:
	 *       - daemon:false
	 *       - name
	 *     cmds:
	 *       - echo "daemon is $FLAG_DAEMON"
	 *       - echo "name is $FLAG_NAME"
	 *       - echo "other args are $@"
	 * ```
	 *
	 * and then we can run `act run foo -daemon -name=Bruno arg1 arg2`
	 * and we should see the following printed to the screen:
	 *
	 * ```bash
	 * daemon is true
	 * name is Bruno
	 * other args are arg1 arg2
	 * ```
	 */
	if len(ctx.Act.Flags) > 0 {
		flagSet := flag.NewFlagSet(ctx.Act.Name, flag.ExitOnError)

		flagVals := make(map[string]string)
		boolPtrs := make(map[string]*bool)
		strPtrs := make(map[string]*string)

		for _, flagName := range ctx.Act.Flags {
			parts := strings.Split(flagName, ":")
			name := parts[0]
			nameKey := strcase.ToCamel(fmt.Sprintf("flag_%s", parts[0]))
			var defaultVal string

			if len(parts) > 1 {
				defaultVal = parts[1]
			}

			if defaultVal == "true" || defaultVal == "false" {
				boolVal := defaultVal == "true"
				boolPtrs[nameKey] = flagSet.Bool(name, boolVal, "")
			} else {
				strPtrs[nameKey] = flagSet.String(name, defaultVal, "")
			}
		}

		/**
		 * Parse the incoming args extracting defined flags if user
		 * provided any.
		 */
		flagSet.Parse(ctx.Args)

		for name, ptr := range boolPtrs {
			if *ptr {
				flagVals[name] = "true"
			}
		}

		for name, ptr := range strPtrs {
			flagVals[name] = *ptr
		}

		// Set cli flags to act ctx.
		ctx.FlagVals = flagVals
		ctx.Args = flagSet.Args()
	}

	// Go over each command and execute them in sequence or in parallel.
	wg := sync.WaitGroup{}

	if ctx.Act.Parallel {
		wg.Add(len(ctx.Act.Cmds))
	}

	// Execute all act commands
	CmdsExec(ctx.Act.Cmds, ctx, &wg)

	/**
	 * Wait all commands to finish because acts going to run
	 * sequentially.
	 */
	wg.Wait()
}

//############################################################
// Exported Functions
//############################################################

/**
 * This function going to find an act to run based on the call id
 * user provided.
 */
func FindActCtx(
	actNames []string,
	actFile *actfile.ActFile,
	prevCtx *ActRunCtx,
	runCtx *RunCtx,
) (*ActRunCtx, error) {
	var targetActName string

	if len(actNames) == 0 {
		targetActName = "_"
	} else {
		targetActName = actNames[0]
	}

	/**
	 * Working directory is always relative to actfile location.
	 */
	wd := path.Dir(actFile.LocationPath)

	/**
	 * If we have a previous matched act context
	 */
	var acts []*actfile.Act
	var actFileLocationPath string

	if prevCtx != nil && prevCtx.Act != nil && len(prevCtx.Act.Acts) > 0 {
		acts = prevCtx.Act.Acts
		actFileLocationPath = prevCtx.ActFile.LocationPath
	} else {
		acts = actFile.Acts
		actFileLocationPath = actFile.LocationPath
	}

	for _, act := range acts {
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

		/**
		 * Now that we macthed an act lets build the run context for it
		 * and start filling it out.
		 */
		ctx := ActRunCtx{
			Act:     act,
			ActFile: actFile,
			PrevCtx: prevCtx,
			Vars:    make(map[string]string),
			ActVars: make(map[string]string),
			RunCtx:  runCtx,
		}

		// Act vars has precedence
		ctx.ActVars["ActName"] = targetActName
		ctx.ActVars["ActFilePath"] = ctx.ActFile.LocationPath
		ctx.ActVars["ActFileDir"] = path.Dir(ctx.ActFile.LocationPath)

		vars := ctx.MergeVars()

		if prevCtx != nil {
			ctx.CallId = strings.Join(append(strings.Split(prevCtx.CallId, ActCallIdSeparator), targetActName), ActCallIdSeparator)
		} else {
			ctx.CallId = targetActName
		}

		utils.LogDebug(fmt.Sprintf("act %s matched with %s in %s", targetActName, act.Name, actFile.LocationPath))

		/**
		 * If we matched an act which contains a `redirect` field defined
		 * then this means we want to forward the execution to
		 * another actfile which contains an act with the same name.
		 * So, for example, if we have the following actfiles:
		 *
		 * ```yaml
		 * # actfile.yml
		 * acts:
		 *   foo:
		 *     redirect: another/actfile.yml
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
		if act.Redirect != "" {
			redirect := utils.CompileTemplate(act.Redirect, vars)
			newActFile := actfile.ReadActFile(utils.ResolvePath(wd, redirect))

			return FindActCtx(actNames, newActFile, &ctx, runCtx)
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
		if act.Include != "" {
			include := utils.CompileTemplate(act.Include, vars)
			newActFile := actfile.ReadActFile(utils.ResolvePath(wd, include))

			return FindActCtx(actNames[1:], newActFile, &ctx, runCtx)
		}

		/**
		 * If act has subacts then lets try to keep matching inside the same
		 * actfile.
		 */
		if len(act.Acts) > 0 && len(actNames) > 0 {
			return FindActCtx(actNames[1:], actFile, &ctx, runCtx)
		}

		return &ctx, nil
	}

	err := errors.New(fmt.Sprintf("act %s not found in %s", targetActName, actFileLocationPath))

	return nil, err
}
