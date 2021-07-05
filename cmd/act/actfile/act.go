/**
 * An Act is an executable unity that users can call by name
 * using act cli.
 */

package actfile

import "gopkg.in/yaml.v3"

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
	 * The first way we can specify what this act going to do
	 * is proving a list of shell commands that going to be
	 * executed in sequence like the following:
	 *
	 * ```yaml
	 * acts:
	 *   foo:
	 *     cmds:
	 *       - echo "im foo"
	 *       - sleep 2
	 *       - echo "im foo again"
	 * ```
	 */
	Cmds []*Cmd

	/**
	 * Another way we can specify the executable part of an act
	 * is providing a path to a shell script file that going to
	 * be invoked when user calls the act. If user specify both
	 * cmds and script then script going to be used.
	 */
	Script string

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
	 * Run act commands in parallel.
	 */
	Parallel bool

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
func ConvertActsObjectToList(actsNode yaml.Node) []*Act {
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
		Desc     string
		Cmds     []*Cmd
		Flags    []string
		Script   string
		Redirect string
		Acts     yaml.Node
		Include  string
		Quiet    bool
		Parallel bool
		Log      string
		Shell    string
	}

	if err := value.Decode(&actObj); err == nil {
		act.Desc = actObj.Desc
		act.Cmds = actObj.Cmds
		act.Flags = actObj.Flags
		act.Script = actObj.Script
		act.Redirect = actObj.Redirect
		act.Include = actObj.Include
		act.Quiet = actObj.Quiet
		act.Parallel = actObj.Parallel
		act.Log = actObj.Log
		act.Shell = actObj.Shell

		/**
		 * Now lets convert acts from map (yaml) to
		 * array (struct) so we can keep acts order.
		 */
		act.Acts = ConvertActsObjectToList(actObj.Acts)
	}

	/**
	 * We can encode act cmds as a simple string of content.
	 */
	var actObj2 struct {
		Desc     string
		Cmds     string
		Flags    []string
		Script   string
		Redirect string
		Acts     yaml.Node
		Include  string
		Quiet    bool
		Parallel bool
		Log      string
		Shell    string
	}

	if err := value.Decode(&actObj2); err == nil {
		cmd := Cmd{
			Cmd: actObj2.Cmds,
		}

		act.Desc = actObj2.Desc
		act.Cmds = []*Cmd{&cmd}
		act.Flags = actObj2.Flags
		act.Script = actObj2.Script
		act.Redirect = actObj2.Redirect
		act.Include = actObj2.Include
		act.Quiet = actObj2.Quiet
		act.Parallel = actObj2.Parallel
		act.Log = actObj2.Log
		act.Shell = actObj2.Shell

		/**
		 * Now lets convert acts from map (yaml) to
		 * array (struct) so we can keep acts order.
		 */
		act.Acts = ConvertActsObjectToList(actObj2.Acts)
	}

	return nil
}
