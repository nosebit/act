/**
 * Command is the smallest unit of execution in any act. An
 * act can be composed by one or more commands that going to
 * be executed in sequence.
 */

package actfile

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

//############################################################
// Types
//############################################################

/**
 * This structure specify a loop for creating multiple similar
 * commands at once.
 */
type CmdLoop struct {
	/**
	 * Specify a list of items to be used in the loop.
	 */
	Items []string

	/**
	 * Create items based with a list of files that match specific
	 * glob pattern.
	 */
	Glob string
}


/**
 * The command struct going to contain everything required for
 * the execution of the command.
 */
type Cmd struct {

	/**
	 * This is the shell command text line that going to be
	 * executed. We use the same name as the struct because
	 * this way users can specify commands like the following:
	 *
	 * ```yaml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "this is a command written as a text line"
	 *       - cmd: echo "this is a command written as an object"
	 * ```
	 *
	 * i.e., we can specify a command as a simple line of text
	 * or as an object full of options. When parsing the yaml
	 * file we going to convert the text line format to object
	 * format.
	 */
	Cmd string

	/**
	 * Another way to specify a command is pointing to a script
	 * file that going to be executed when we execute this
	 * command. This way we can have something like this:
	 *
	 * ```yaml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "text line command format"
	 *       - cmd: echo "object command format"
	 *       - script: path/to/script.sh
	 *       - script: path/to/another/script.sh
	 * ```
	 *
	 * and this way we can have a mix of simple commands specified
	 * as simple lines of text and commands that invoke script
	 * which can implement really complex logic.
	 */
	Script string

	/**
	 * Set the shell to be used when running commands. By default
	 * we use bash shell.
	 */
	Shell string

	/**
	 * A command can reference another act to run like this:
	 *
	 * ```yaml
	 * # actfile.yml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "foo before bar"
	 *       - act: bar
	 *       - echo "foo after bar"
	 *   bar:
	 *     cmds:
	 *       - echo "im bar"
	 * ```
	 *
	 * so when we run `act run foo` we going to see this printed:
	 *
	 * ```bash
	 * foo before bar
	 * im bar
	 * foo after bar
	 * ```
	 */
	Act string

	/**
	 * When running an act we can specify the actfile from where
	 * to get the act file.
	 */
	From string

	/**
	 * When running an act we can specify if we want to run it
	 * as a detached process.
	 */
	Detach bool

	/**
	 * With this we can create loops for executing multiple similar
	 * commands.
	 */
	Loop *CmdLoop

	/**
	 * This flag allows mismatching act (skiping not found error).
	 */
	Mismatch string

	/**
	 * List of command line arguments to pass over to cmd/act when
	 * executing it.
	 */
	Args []string

	/**
	 * Disable logging
	 */
	Quiet bool

	/**
	 * Enable or disable log.
	 */
	Log bool
}

//############################################################
// Cmd Struct Functions
//
// Learning Notes: This is more or less the way we can have
// object orientation in go. All functions defined like the
// following going to be available for struct instances.
//############################################################

/**
 * This function implements the unmarshal interface of go-yaml
 * module so commands can be correctly parsed from actfile.yaml
 * file. The idea here is to correctly produce Cmd structs from
 * what we get from actfile.yml. As we pointed in the comments
 * for the Cmd struct we can have some "polimorphic" format of
 * commands in actfile.yml. It can be a simple single line of
 * text or it can be an object for instance. This function going
 * to handle this different scenarios and generate a correct
 * Cmd struct.
 */
func (cmd *Cmd) UnmarshalYAML(value *yaml.Node) error {
	/**
	 * First the more often case: we try to parse a command comming
	 * from yaml file as a simple single line of text.
	 */
	var cmdLine string

	if err := value.Decode(&cmdLine); err == nil {
		/**
		 * We were able to correctly parse the command as a string
		 * from yaml file so we fulfill our cmd accordingly and
		 * return.
		 */
		cmd.Cmd = cmdLine
		return nil
	}

	/**
	 * Otherwise if we couldn't parse command as a simple string
	 * from yaml file then we try to parse it as an object with
	 * some specific fields. In this case the object is the same
	 * as Cmd struct but it could be different.
	 */
	var cmdObj struct {
		Cmd    		string
		Script 		string
		Shell     string
		Act    		string
		From   		string
		Detach 		bool
		Args   		[]string
		Quiet  		bool
		Log  			bool
		Loop   		*CmdLoop
		Mismatch 	string
	}

	if err := value.Decode(&cmdObj); err == nil {
		cmd.Cmd = cmdObj.Cmd
		cmd.Script = cmdObj.Script
		cmd.Shell = cmdObj.Shell
		cmd.Act = cmdObj.Act
		cmd.From = cmdObj.From
		cmd.Detach = cmdObj.Detach
		cmd.Args = cmdObj.Args
		cmd.Quiet = cmdObj.Quiet
		cmd.Log = cmdObj.Log
		cmd.Loop = cmdObj.Loop
		cmd.Mismatch = cmdObj.Mismatch

		// We let user pass command args together with act name.
		if cmdObj.Act != "" {
			args := strings.Split(cmdObj.Act, " ")
			actCallId := args[0]
			actArgs := args[1:]

			cmd.Act = actCallId
			cmd.Args = append(cmd.Args, actArgs...)
		}

		// We let user pass command args together with script.
		if cmdObj.Script != "" {
			// Trim whitespaces from template strings
			var re = regexp.MustCompile(`{{ *([^ ]+) *}}`)
			scriptLine := re.ReplaceAllString(cmdObj.Script, "{{$1}}")

			args := strings.Split(scriptLine, " ")
			scriptPath := args[0]
			scriptArgs := args[1:]

			cmd.Script = scriptPath
			cmd.Args = append(cmd.Args, scriptArgs...)
		}

		return nil
	}

	return nil
}
