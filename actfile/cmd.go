/**
 * Command is the smallest unit of execution in any act. An
 * act can be composed by one or more commands that going to
 * be executed in sequence.
 */

package actfile

import (
	"strings"
	
	"gopkg.in/yaml.v3"
)

//############################################################
// Types
//############################################################

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
	 * List of command line arguments to pass over to cmd/act when
	 * executing it.
	 */
	Args []string
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
		Cmd    string
		Script string
		Act    string
		Args   []string
	}

	if err := value.Decode(&cmdObj); err == nil {
		cmd.Cmd = cmdObj.Cmd
		cmd.Script = cmdObj.Script
		cmd.Act = cmdObj.Act
		cmd.Args = cmdObj.Args

		// We let user pass command args together with act name.
		if cmdObj.Act != "" {
			args := strings.Split(cmdObj.Act, " ")
			actCallId := args[0]
			actArgs := args[1:]

			cmd.Act = actCallId
			cmd.Args = append(cmd.Args, actArgs...)
		}

		return nil
	}

	return nil
}
