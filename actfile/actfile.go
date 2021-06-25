/**
 * The actfile going to specify all acts an user can
 * invoke using act cli and the execution context (like global
 * vars etc).
 */

package actfile

import (
	"os"
	"sync"

	"github.com/nosebit/act/utils"
	"gopkg.in/yaml.v3"
)

//############################################################
// Types
//############################################################
/**
 * This is the main struct that we going to fulfill with data
 * comming from actfile.yml config file.
 */
type ActFile struct {

	/**
	 * The actfile version going to guide us regarding the
	 * structure of fields we have in actfile.yml file.
	 */
	Version string

	/**
	 * Actfile namespace for logging
	 */
	Namespace string

	/**
	 * This is a list of commands to be run before execution
	 * of any act.
	 */
	BeforeAll *Act

	/**
	 * The user specifies one or more acts in the actfile. Each
	 * act is a executable unit the user can call by name
	 * using the cli command `act run <actName>`. Acts are
	 * specified as a key value map where the key is the act
	 * name and the value is the act specification.
	 */
	Acts []*Act

	/**
	 * This is the actfile location path in file system.
	 */
	LocationPath string

	/**
	 * Env is a dotenv file we going to load before running
	 * any act.
	 */
	EnvFilePath string

	/**
	 * This wait groups tell parallels acts that actfile
	 * was initialized.
	 */
	InitWg *sync.WaitGroup
}

//############################################################
// Actfile Struct Functions
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
func (actFile *ActFile) UnmarshalYAML(value *yaml.Node) error {
	var actFileObj struct {
		Version   		string
		Namespace 		string
		BeforeAll 		*Act `yaml:"before-all"`
		Acts      		yaml.Node
		EnvFilePath   string `yaml:"envfile"`
	}

	if err := value.Decode(&actFileObj); err == nil {
		actFile.Version = actFileObj.Version
		actFile.Namespace = actFileObj.Namespace
		actFile.BeforeAll = actFileObj.BeforeAll
		actFile.EnvFilePath = actFileObj.EnvFilePath

		if actFile.BeforeAll != nil {
			actFile.BeforeAll.Name = "before"
		}

		var acts []*Act

		for i := 0; i < len(actFileObj.Acts.Content); i += 2 {
			var actName string
			var act Act

			actFileObj.Acts.Content[i].Decode(&actName)
			actFileObj.Acts.Content[i+1].Decode(&act)

			act.Name = actName

			acts = append(acts, &act)
		}

		actFile.Acts = acts
	}

	return nil
}

//############################################################
// Exposed Functions
//
// Learning Note: In go exposed props (like in structs) and
// exposed functions should start with a capital letter. Props
// and functions starting with lowercase are private to the
// package
//############################################################

/**
 * This function going to read/parse and actfile.yml from a
 * specific directory.
 */
func ReadActFile(filepath string) *ActFile {
	/**
	 * We start by creating an empty Actfile struct so we can
	 * fulfill it.
	 */
	spec := ActFile{}

	// Try to open actfile.yml
	file, err := os.Open(filepath)

	/**
	 * If we can't open the file (it does not exists for example)
	 * then we give up.
	 */
	if err != nil {
		utils.FatalError("could not read actfile", err)
	}

	// Parse yaml file
	yaml.NewDecoder(file).Decode(&spec)

	// Set location path
	spec.LocationPath = filepath

	/**
	 * @TODO : shouldn't we handle yaml parse errors here??
	 */

	return &spec
}
