/**
 * This file going to implement the list subcommand which
 * is responsible for listing all running acts.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/nosebit/act/actfile"
	"github.com/nosebit/act/utils"
)

//############################################################
// Types
//############################################################

/**
 * This struct going to hold some execution info we going to
 * collect from running acts.
 */
type ActRunInfo struct {
	/**
	 * This uniquely identifies the act we are running and it's
	 * an MD5 hash of act call id inside the tail actfile (i.e.,
	 * the actfile we actually found the act to run) together with
	 * the actfile location path. So it's a hash of something like
	 * "/path/to/actfile.yml:foo.bar" for a bar subact of foo act.
	 */
	ActId string

	/**
	 * The act name which going to be the last act name we see
	 * in ActNameId. If this act is a subact that was called
	 * with `act run foo.bar` then ActName going to be `bar`.
	 */
	ActName string

	/**
	 * This is the full act name "path" which uniqueky identifies
	 * the call to execute the act. If the act is a subact that
	 * was called with `act run foo.bar` then ActCallId is `foo.bar`.
	 */
	ActCallId string

	/**
	 * This is the process id for the running act.
	 */
	Pid int

	/**
	 * This is the process groupd id running act and all it's
	 * children processes.
	 */
	Pgid int

	/**
	 * This is the path for the log file.
	 */
	LogFilePath string
}

//############################################################
// Exposed Functions
//############################################################

/**
 * This function going to save an act run info to a json file
 * in the propert act data dir.
 */
func SaveActRunInfo(info *ActRunInfo) {
	file, _ := json.MarshalIndent(info, "", " ")

	actDataDirPath := utils.GetActDataDirPath(info.ActCallId)

	os.MkdirAll(actDataDirPath, 0755)

	infoFilePath := path.Join(actDataDirPath, "info.json")

	if err := ioutil.WriteFile(infoFilePath, file, 0644); err != nil {
		utils.FatalError("could not save act run info file", err)
	}
}

/**
 * This function reads an act run info from json.
 */
func ReadActRunInfoFromJson(jsonPath string) *ActRunInfo {
	file, err := os.Open(jsonPath)

	if err != nil {
		utils.FatalError("could not read act info file", err)
	}

	defer file.Close()

	fileContent, _ := ioutil.ReadAll(file)

	var info ActRunInfo

	json.Unmarshal(fileContent, &info)

	info.LogFilePath = path.Join(path.Dir(jsonPath), "log")

	return &info
}

/**
 * This function going to find run info for a particular act.
 *
 * @param actCallId - Id that uniquely identifies the act
 *   in the form foo1.foo2...fooN.
 */
func GetActRunInfo(actCallId string) *ActRunInfo {
	infoFilePath := path.Join(utils.GetActDataDirPath(actCallId), "info.json")

	if _, err := os.Stat(infoFilePath); err == nil {
		return ReadActRunInfoFromJson(infoFilePath)
	}

	utils.FatalError("no act found")

	return nil
}

/**
 * This function going to find info for all currently running
 * acts. For this to happen we going to traverse the whole
 * directory tree inside data dir.
 */
func GetAllActRunInfo() []*ActRunInfo {

	var infos []*ActRunInfo

	var readRecursive func(dirPath string)

	readRecursive = func(dirPath string) {
		files, err := ioutil.ReadDir(dirPath)

		if err != nil {
			utils.FatalError("could not react act dir", err)
		}

		for _, f := range files {
			newPath := path.Join(dirPath, f.Name())

			if f.IsDir() {
				readRecursive(newPath)
			} else if f.Name() == "info.json" {
				info := ReadActRunInfoFromJson(newPath)
				infos = append(infos, info)
			}
		}
	}

	readRecursive(utils.GetDataDirPath())

	return infos
}

/**
 * This is the main execution point for the `list` command.
 */
func ListCmdExec(_ []string, _ *actfile.ActFile) {
	infos := GetAllActRunInfo()

	for _, info := range infos {
		fmt.Println(info.ActCallId)
	}
}
