package run

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"syscall"

	"github.com/logrusorgru/aurora/v3"
	"github.com/nosebit/act/cmd/act/utils"
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
	Id string

	/**
	 * If this act was created from another act process then we
	 * going to store parent act id here. We do this because we
	 * need to update parent when the state of this act change.
	 */
	ParentActId string

	/**
	 * Name is a human friendly id assigned by the user when
	 * running the act. User can then use this name to stop
	 * o get logs for the act.
	 */
	NameId string

	/**
	 * This is the process group id of this act process.
	 */
	Pgid int

	/**
	 * This is the process id.
	 */
	Pid int

	/**
	 * This is the list of all command process group ids created
	 * by this act process. When we are running a sync act then
	 * at any given time this array going to have one and only one
	 * pgid (the pgid of currently running command). When running a
	 * parallel act then usually this array going to contain the
	 * pgids of all commands running in parallel.
	 */
	CmdPgids []int

	/**
	 * This is a list of ids of all act detached processes created
	 * by this act process.
	 */
	ChildActIds []string

	/**
	 * Flag to indicate we are killing the process.
	 */
	IsKilling bool

	/**
	 * Mutex to pevent race conditions of multiple parallel
	 * commands changing the same info struct.
	 */
	mutex sync.Mutex `json:"-"`
}

//############################################################
// Info Struct Functions
//############################################################
/**
 * This function going to add a new child act run id to info
 * and then save info back to file system.
 */
func (info *Info) AddChildActId(id string) {
	info.mutex.Lock()

	idx := -1

	for i, val := range info.ChildActIds {
		if val == id {
			idx = i
			break
		}
	}

	if idx < 0 {
		info.ChildActIds = append(info.ChildActIds, id)
		info.Save()
	}

	info.mutex.Unlock()
}

/**
 * This function removes a child act run id from info and
 * then save the info back to file system.
 */
func (info *Info) RmChildActId(id string) {
	info.mutex.Lock()

	idx := -1

	for i, val := range info.ChildActIds {
		if val == id {
			idx = i
			break
		}
	}

	if idx >= 0 {
		info.ChildActIds = append(info.ChildActIds[:idx], info.ChildActIds[idx+1:]...)
		info.Save()
	}

	info.mutex.Unlock()
}

/**
 * This function going to add a new Pgid to info and then save
 * info back to file system.
 */
func (info *Info) AddCmdPgid(pgid int) {
	info.mutex.Lock()

	idx := -1

	for i, val := range info.CmdPgids {
		if val == pgid {
			idx = i
			break
		}
	}

	if idx < 0 {
		info.CmdPgids = append(info.CmdPgids, pgid)
		info.Save()
	}

	info.mutex.Unlock()
}

/**
 * This function removes a pgid from info and then save the info
 * back to file system.
 */
func (info *Info) RmCmdPgid(pgid int) {
	info.mutex.Lock()

	idx := -1

	for i, val := range info.CmdPgids {
		if val == pgid {
			idx = i
			break
		}
	}

	if idx >= 0 {
		info.CmdPgids = append(info.CmdPgids[:idx], info.CmdPgids[idx+1:]...)
		info.Save()
	}

	info.mutex.Unlock()
}

/**
 * This function going to set IsKilling flag.
 */
func (info *Info) SetIsKilling() {
	info.mutex.Lock()

	info.IsKilling = true
	info.Save()

	info.mutex.Unlock()
}

/**
 * This function get name id if present or id otherwise.
 */
func (info *Info) GetNameIdOrId() string {
	if info.NameId != "" {
		return info.NameId
	}

	return info.Id
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

	os.RemoveAll(dataDirPath)
}

/**
 * This function going to quit a running process associated
 * with this specific info.
 */
func (info *Info) Kill() {
	/**
	 * To prevent child acts killing this process we going to add a
	 * fake pgid to running pgids.
	 */
	info.SetIsKilling()

	/**
	 * Kill all child acts.
	 */
	if len(info.ChildActIds) > 0 {
		for _, childId := range info.ChildActIds {
			childInfo := GetInfo(childId)

			if childInfo != nil {
				childInfo.Kill()
			}
		}
	}

	// Kill all running commands.
	for _, pgid := range info.CmdPgids {
		if pgid < 0 {
			continue
		}

		if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
			utils.FatalError(fmt.Sprintf("could not kill command with process pgid=%d", pgid), err)
		}
	}

	// Remove data dir
	info.RmDataDir()

	// Print
	fmt.Println(fmt.Sprintf("act %s stopped", aurora.Green(info.GetNameIdOrId()).Bold()))

	// Kill parent if needed
	if info.ParentActId != "" {
		parentInfo := GetInfo(info.ParentActId)

		if parentInfo != nil && !parentInfo.IsKilling {
			// Remove from parent
			parentInfo.RmChildActId(info.Id)

			// If parent is still running something then we finish.
			if len(parentInfo.CmdPgids) > 0 || len(parentInfo.ChildActIds) > 0 {
				return
			}

			parentInfo.Kill()
		}
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
	if _, err := os.Stat(jsonPath); err == nil {
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

	return nil
}

//############################################################
// Exported Functions
//############################################################
/**
 * This function get call stack from an act id.
 */
func GetInfoCallStack(id string) []*Info {
	allInfos := GetAllInfo()

	// Convert to map for simplicity
	infoMap := make(map[string]*Info)

	for _, info := range allInfos {
		infoMap[info.Id] = info
	}

	var stack []*Info
	info, hasInfo := infoMap[id]

	for hasInfo {
		stack = append([]*Info{info}, stack...)

		if info.ParentActId != "" {
			info, hasInfo = infoMap[info.ParentActId]
		} else {
			hasInfo = false
		}
	}

	return stack
}

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
			dirPath := path.Join(dataDirPath, f.Name())
			jsonPath := path.Join(dirPath, InfoFileName)
			info := loadInfoFromFile(jsonPath)

			if info == nil {
				// Remove folder
				os.RemoveAll(dirPath)
			} else {
				infos = append(infos, info)
			}
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
			dirPath := path.Join(dataDirPath, f.Name())
			jsonPath := path.Join(dirPath, InfoFileName)
			info := loadInfoFromFile(jsonPath)

			if info == nil {
				// Remove folder
				os.RemoveAll(dirPath)
			} else if info.NameId == name || info.Id == name {
				return info
			}
		}
	}

	return nil
}
