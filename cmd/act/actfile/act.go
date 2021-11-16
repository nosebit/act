/**
 * An Act is an executable unity that users can call by name
 * using act cli.
 */

package actfile

import (
	"gopkg.in/yaml.v3"
)

//############################################################
// Types
//############################################################

/**
 * Acts going to be specified in actfile as a key-value map
 * where the key is the act name and value is the act
 * specification.
 */
type ActsMap map[string]*Act

/**
 * Act exec stage.
 */
type ActExecStage struct {
	/**
	 * Act stage name.
	 */
	Name string

	/**
	 * Flag indicating if commands in this stage should be
	 * run in parallel.
	 */
	Parallel bool

	/**
	 * Commands to be executed in this exec stage.
	 */
	Cmds []*Cmd

	/**
	 * Path to a script to run instead of commands.
	 */
	Script string

	/**
	 * Set the shell to be used when running commands. By default
	 * we use bash shell.
	 */
	Shell string

	/**
	 * Prevent logging.
	 */
	Quiet bool

	/**
	 * Flag indicating if this stage is killed.
	 */
	IsKilled bool
}

/**
 * Act check.
 */
type ActCheck struct {
	/**
	 * Commands to run in order to check if act is in success state.
	 */
	Cmds []*Cmd

	/**
	 * Interval to run checks.
	 */
	Interval int
}

/**
 * This is the struct we going to get fulfilled with data
 * coming from actfile.yml file.
 */
type Act struct {
	/**
	 * The is a MD5 hash of act name id inside an actfile
	 * (like `foo.bar` for bar subact of foo act) and the
	 * actfile location path. This way we can uniquely identify
	 * the act when looking different actfiles.
	 */
	Id string

	/**
	 * The act name is actually a regex which we use to match
	 * against act name provided by user during run call. So
	 * suppose we have:
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   foo-.+:
	 *     cmds:
	 *       - echo "helo foo stuff"
	 * ```
	 * the act name is "foo-.+" but it's going to be executed
	 * when user runs `act run foo-world` for example.
	 */
	Name string

	/**
	 * Act call id is how we uniquely identify an act in a
	 * subact chain. So, suppose we have the following:
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   foo:
	 *     acts:
	 *       bar:
	 *         cmds:
	 *            - echo "im foo bar subact"
	 * ```
	 *
	 * and we call `act run foo.bar` to execute the bar subact
	 * of foo act. Then `foo.bar` is the call id while
	 * `bar` is the act name.
	 */
	CallId string

	/**
	 * A textual description about the act which going to be
	 * used in the help command to give user a guess about
	 * what the act do.
	 */
	Desc string

	/**
	 * List of CLI flags that can be passed over to this act.
	 */
	Flags []string

	/**
	 * Info about how to check if act is in success state
	 * (useful for long running acts).
	 */
	Check *ActCheck

	/**
	 * Location of a file containing env vars we should load when
	 * running this act.
	 */
	EnvFilePath string

	/**
	 * Definition for act start exec stage. This is the main
	 * exec stage and is the only required one. User can define
	 * this stage in the following ways in actfile:
	 *
	 * ```yaml
	 * # Using cmds (deprecated)
	 * acts:
	 *   foo:
	 *     parallel: true
	 *     cmds:
	 *       - echo "im foo"
	 *       - sleep 2
	 *       - echo "im foo again"
	 * ```
	 *
	 * ```yaml
	 * # Using start (sequentially)
	 * acts:
	 *   foo:
	 *     start:
	 *       - echo "im foo"
	 *       - sleep 2
	 *       - echo "im foo again"
	 * ```
	 *
	 * ```yaml
	 * # Using start (parallel)
	 * acts:
	 *   foo:
	 *     start:
	 *       parallel: true
	 *       cmds:
	 *         - echo "im foo"
	 *         - sleep 2
	 *         - echo "im foo again"
	 * ```
	 *
	 * ```yaml
	 * # Using start (simple command)
	 * acts:
	 *   foo:
	 *     start: echo "hello"
	 * ```
	 */
	Start *ActExecStage

	/**
	 * Definition for act before exec stage. Commands in
	 * this stage going to be executed just before executing
	 * the start stage.
	 */
	Before *ActExecStage

	/**
	 * Definition for act after exec stage. Commands in
	 * this stage going to be executed when the act is
	 * in success state (via check commands).
	 */
	After *ActExecStage

	/**
	 * @deprecated - use Final stage instead
	 *
	 * Definition for act teardown exec stage. Commands
	 * in this stage going to be executed just before
	 * exiting the main program (due to success or error).
	 */
	Teardown *ActExecStage

	/**
	 * This stage going to be executed just before exiting the
	 * executing. This is always going to be called no matter
	 * the main act succedded or failed.
	 */
	Final *ActExecStage

	/**
	 * If we want to reuse an action with same name located in
	 * another actfile then we can specify this another actfile
	 * file path in this field. So if we have:
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
	 *       - echo "im foo"
	 * ```
	 *
	 * then when we invoke `act run foo` in the folder containing
	 * actfile.yml we going to get "im foo" printed in the screen.
	 */
	Redirect string

	/**
	 * We can specify nested acts that can be invoked like sub
	 * commands of the main act. For example, if we have
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "im foo"
	 *     acts:
	 *       bar:
	 *         cmds:
	 *           - echo "im bar"
	 * ```
	 *
	 * then we can invoke bar sub act using `act run foo bar`
	 */
	Acts []*Act

	/**
	 * Another way to place sub/nested acts is including all acts
	 * from another actfile as sub acts. So lets say we have
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "im foo"
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
	 *       - echo "im bar"
	 * ```
	 *
	 * then we can still invoke bar using `act run foo bar`. This
	 * allows us to split act definition in multiple files.
	 */
	Include string

	/**
	 * Prevent logging.
	 */
	Quiet bool

	/**
	 * Log mode.
	 */
	Log string

	/**
	 * Set the shell to be used when running commands. By default
	 * we use bash shell.
	 */
	Shell string
}

//############################################################
// Internal Functions
//############################################################
/**
 * This function going to receive a generic yaml node representing
 * the acts map and convert it to an array of acts so we can
 * keep the same key order of the defined map by user.
 */
func DecodeActs(actsNode yaml.Node) []*Act {
	var acts []*Act

	for i := 0; i < len(actsNode.Content); i += 2 {
		var actName string
		var act Act

		actsNode.Content[i].Decode(&actName)
		actsNode.Content[i+1].Decode(&act)

		act.Name = actName

		acts = append(acts, &act)
	}

	return acts
}

/**
 * This function going to decode generic cmds.
 */
func DecodeCmds(cmdsNode yaml.Node) []*Cmd {
	/**
	 * Try to decode from string first, then directly
	 * from array.
	 */
	var cmds []*Cmd
	var cmdStr string

	if err := cmdsNode.Decode(&cmdStr); err == nil {
		// For some reason if we don't have cmdNode its decoding to string.
		if cmdStr == "" {
			return nil
		}

		cmd := &Cmd{Cmd: cmdStr}
		cmds = append(cmds, cmd)
		return cmds
	} else if err := cmdsNode.Decode(&cmds); err == nil {
		return cmds
	}

	return nil
}

/**
 * This function going to convert generic exec stages.
 */
func DecodeExecStage(stageNode yaml.Node, name string) *ActExecStage {
	var stageObj struct {
		Name     string
		Parallel bool
		Cmds     yaml.Node
		Script   string
		Shell    string
		Quiet    bool
	}

	/**
	 * Try to decode stage as string first, then as an array,
	 * and then as an map.
	 */
	var stageStr string
	var stageArr []*Cmd

	if err := stageNode.Decode(&stageStr); err == nil {
		// For some reason if we don't have stageNode its decoding to string.
		if stageStr == "" {
			return nil
		}

		cmd := &Cmd{Cmd: stageStr}

		return &ActExecStage{
			Name: name,
			Cmds: []*Cmd{cmd},
		}
	} else if err := stageNode.Decode(&stageArr); err == nil {
		return &ActExecStage{
			Name: name,
			Cmds: stageArr,
		}
	} else if err := stageNode.Decode(&stageObj); err == nil {
		cmds := DecodeCmds(stageObj.Cmds)

		if cmds != nil {
			return &ActExecStage{
				Name:     name,
				Parallel: stageObj.Parallel,
				Cmds:     cmds,
				Script:   stageObj.Script,
				Shell:    stageObj.Shell,
				Quiet:    stageObj.Quiet,
			}
		}
	}

	return nil
}

//############################################################
// Act Struct Functions
//
// Learning Notes: This is more or less the way we can have
// object orientation in go. All functions defined like the
// following going to be available for struct instances.
//############################################################

/**
 * This function instructs yaml how to correctly parse actfile
 * from yaml file. We basically needs this here to convert acts
 * from map (in yaml file) to array (in struct) so we can preserve
 * the order of acts as defined in the yaml file. This is
 * important because we need order to correctly match act name
 * (i.e., acts defined first has precedence during matching).
 */
func (act *Act) UnmarshalYAML(value *yaml.Node) error {
	var actObj struct {
		Desc   				string
		Cmds    			yaml.Node
		Flags    			[]string
		Script   			string
		Redirect 			string
		Acts     			yaml.Node
		Include  			string
		Quiet    			bool
		Parallel 			bool
		Log      			string
		Shell    			string
		EnvFilePath 	string `yaml:"envfile"`
		Before   			yaml.Node
		Start    			yaml.Node
		After    			yaml.Node
		Final 				yaml.Node
		Teardown 			yaml.Node
	}

	if err := value.Decode(&actObj); err == nil {
		act.Desc = actObj.Desc
		act.Flags = actObj.Flags
		act.EnvFilePath = actObj.EnvFilePath
		act.Redirect = actObj.Redirect
		act.Include = actObj.Include
		act.Quiet = actObj.Quiet
		act.Log = actObj.Log
		act.Shell = actObj.Shell

		// Lets decode fields
		act.Acts = DecodeActs(actObj.Acts)

		// Decode start stage
		act.Start = DecodeExecStage(actObj.Start, "start")
		cmds := DecodeCmds(actObj.Cmds)

		if act.Start == nil && cmds != nil {
			act.Start = &ActExecStage{
				Name:     "start",
				Cmds:     cmds,
				Parallel: actObj.Parallel,
				Script:   actObj.Script,
			}
		}

		act.Before = DecodeExecStage(actObj.Before, "before")
		act.After = DecodeExecStage(actObj.After, "after")
		act.Final = DecodeExecStage(actObj.Final, "final")

		// @deprecated
		act.Teardown = DecodeExecStage(actObj.Teardown, "final")
	}

	return nil
}
