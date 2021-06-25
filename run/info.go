package run

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/nosebit/act/utils"
)

//############################################################
// Exported Constants
//############################################################

/**
 * In s subact chain we separate each act name by this separator
 * like when we run `act run foo.bar`. In this case `bar` is a
 * subact of the act `foo` and the whole act is uniquely identified
 * by the name `foo.bar`.
 */
const ActCallIdSeparator = "."

/**
 * This is the name of the directory where we going to hold
 * all info for running acts.
 */
const ActDataDirName = ".actdt"

/**
 * This is the file name we going to use when saving the info
 * struct back to file system.
 */
const InfoFileName = "info.json"

/**
 * This is the name of dotenv var file we use to share variables
 * between act/command execution.
 */
const EnvFileName = "env"

//############################################################
// Types
//############################################################

/**
 * This struct going to hold run context info that going to
 * be stored to a file describing a running act.
 */
type Info struct {
	/**
	 * Which running act going to receive an unique short hash
	 * id which going to be used to name the data folder for
	 * this act in the act data dir.
	 */
	Id    string

	/**
	 * If this act was created from another act process then we
	 * going to store parent act id here. We do this because we
	 * need to update parent when the state of this act change.
	 */
	ParentId string

	/**
	 * Name is a human friendly id assigned by the user when
	 * running the act. User can then use this name to stop
	 * o get logs for the act.
	 */
	NameId  string

	/**
	 * List of all process group ids of spawned commands. We
	 * use this when we need to stop/kill a running act.
	 */
	Pgids []int
}

//############################################################
// Info Struct Functions
//############################################################
/**
 * This function going to add a new Pgid to info and then save
 * info back to file system.
 */
func (info *Info) AddPgid(pgid int) {
	info.Pgids = append(info.Pgids, pgid)
	info.Save()
}

/**
 * This function removes a pgid from info and then save the info
 * back to file system.
 */
func (info *Info) RmPgid(pgid int) {
	idx := -1

	for i, val := range info.Pgids {
		if val == pgid {
			idx = i
			break
		}
	}

	if idx >= 0 {
		info.Pgids = append(info.Pgids[:idx], info.Pgids[idx+1:]...)
		info.Save()
	}
}

/**
 * This function get data dir for this run info.
 */
func (info *Info) GetDataDirPath() string {
	return path.Join(utils.GetWd(), ActDataDirName, info.Id)
}

/**
 * This function get the log file path for this run info.
 */
func (info *Info) GetLogFilePath() string {
	return path.Join(utils.GetWd(), ActDataDirName, info.Id, "log")
}

/**
 * This function get env vars file path for this run info.
 */
func (info *Info) GetEnvVarsFilePath() string {
	return path.Join(info.GetDataDirPath(), EnvFileName)
}

/**
 * This function going to save info to a file in the data
 * directory.
 */
func (info *Info) Save() {
	content, _ := json.MarshalIndent(info, "", " ")

	dirPath := info.GetDataDirPath()

	os.MkdirAll(dirPath, 0755)

	infoFilePath := path.Join(dirPath, InfoFileName)

	if err := ioutil.WriteFile(infoFilePath, content, 0644); err != nil {
		utils.FatalError("could not save run info file", err)
	}
}

/**
 * This function going to remove run info directory.
 */
func (info *Info) RmDataDir() {
	dataDirPath := info.GetDataDirPath()

	err := os.RemoveAll(dataDirPath)

	if err != nil {
		utils.FatalError(fmt.Sprintf("could not remove dir %s", dataDirPath), err)
	}
}

//############################################################
// Internal Functions
//############################################################
/**
 * This function going to read an info struct from the data folder
 * directory. We receive the path to json representing the info
 * struct and then we fill the struct with content of the file.
 */
func loadInfoFromFile(jsonPath string) *Info {
	file, err := os.Open(jsonPath)

	if err != nil {
		utils.FatalError("could not read act info file", err)
	}

	defer file.Close()

	fileContent, _ := ioutil.ReadAll(file)

	var info Info

	json.Unmarshal(fileContent, &info)

	return &info
}

//############################################################
// Exported Functions
//############################################################

/**
 * This function going to get all run info.
 */
func GetAllInfo() []*Info {
	dataDirPath := path.Join(utils.GetWd(), ActDataDirName)

	files, err := ioutil.ReadDir(dataDirPath)
	var infos []*Info

	if err != nil {
		utils.FatalError("could not react act dir", err)
	}

	for _, f := range files {
		if f.IsDir() {
			jsonPath := path.Join(dataDirPath, f.Name(), InfoFileName)
			info := loadInfoFromFile(jsonPath)

			infos = append(infos, info)
		}
	}

	return infos
}

/**
 * This function get info for a specific act by its name
 * as associated by the user.
 */
func GetInfo(name string) *Info {
	dataDirPath := path.Join(utils.GetWd(), ActDataDirName)

	files, err := ioutil.ReadDir(dataDirPath)

	if err != nil {
		utils.FatalError("could not react act dir", err)
	}

	for _, f := range files {
		if f.IsDir() {
			jsonPath := path.Join(dataDirPath, f.Name(), InfoFileName)
			info := loadInfoFromFile(jsonPath)

			if info.NameId == name || info.Id == name {
				return info
			}
		}
	}

	return nil
}
