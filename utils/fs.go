/**
 * This file expose functions to handle file system operations.
 */

package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

//############################################################
// Constants
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
const DataDirName = ".actdt"

//############################################################
// Exposed Functions
//############################################################
/**
 * This function going to get the working directory from
 * which we invoked the act binary. As you can see the main
 * utility of this function is to properly handle error.
 */
func GetWd() string {
	dir, err := os.Getwd()

	if err != nil {
		FatalError("could not get working directory", err)
	}

	return dir
}

/**
 * This function going to check if a file exists.
 *
 * @param filepath - Absolute path of the file to check.
 */
func DoFileExists(filepath string) bool {
	if _, err := os.Stat(filepath); err == nil {
		return true
	}

	return false
}

/**
 * This function going to check if a folder is empty.
 */
func IsEmptyDir(dirPath string) bool {
	dir, err := os.Open(dirPath)

	if err != nil {
		FatalError("could not open dir to check if its empty", err)
	}

	/**
	 * Using defer so this function gets called when the caller
	 * function returns.
	 */
	defer dir.Close()

	_, err = dir.Readdir(1)

	return err == io.EOF
}

/**
 * This function going to remove a directory and all of its empty
 * ancestor directories.
 *
 * @param dirPath - The starting directory path.
 * @param stopDirName - Directory which should be used as a stop
 *   point for the remove process. When we find a directory with
 *   this name we going to prevent removing it and all it's
 *   ancestors.
 */
func RmDirAndEmptyAncestors(dirPath string, stopDirName string) {
	err := os.RemoveAll(dirPath)

	if err != nil {
		FatalError(fmt.Sprintf("could not remove dir %s", dirPath), err)
	}

	parentDirPath := path.Dir(dirPath)
	parentDirName := path.Base(parentDirPath)

	for parentDirName != stopDirName && IsEmptyDir(parentDirPath) {
		err := os.RemoveAll(parentDirPath)

		if err != nil {
			FatalError(fmt.Sprintf("could not remove dir %s", parentDirPath), err)
		}

		parentDirPath = path.Dir(parentDirPath)
		parentDirName = path.Base(parentDirPath)
	}
}

/**
 * This function going to remove an act data dir.
 */
func RmActDataDir(actCallId string) {
	dirPath := GetActDataDirPath(actCallId)

	RmDirAndEmptyAncestors(dirPath, DataDirName)
}

/**
 * This function going to write a simple string to a file in append
 * mode.
 */
func WriteToFile(filePath string, text string) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		FatalError(fmt.Sprintf("could not open file %s to write", filePath), err)
	}

	if _, err := file.Write([]byte(text)); err != nil {
		FatalError(fmt.Sprintf("could not write to file %s", filePath), err)
	}

	file.Close()
}

/**
 * This function resolves a path relatively to working dir.
 */
func ResolvePathFromWd(aPath string) string {
	return path.Join(GetWd(), aPath)
}

/**
 * This function gets the acts data dir path.
 */
func GetDataDirPath() string {
	return path.Join(GetWd(), DataDirName)
}

/**
 * This function get act data dir path for a specific act
 * name id or sequence of act names.
 */
func GetActDataDirPath(actCallId string) string {
	actNames := strings.Split(actCallId, ActCallIdSeparator)
	actDirPath := path.Join(actNames...)

	return path.Join(GetDataDirPath(), actDirPath)
}

/**
 * This going to get the pid file path for a specific act.
 */
func GetActPidFilePath(actCallId string) string {
	dtdir := GetActDataDirPath(actCallId)

	return path.Join(dtdir, "pid")
}

/**
 * This going to get the log file path for a specific act.
 */
func GetActLogFilePath(actCallId string) string {
	dtdir := GetActDataDirPath(actCallId)

	return path.Join(dtdir, "log")
}

/**
 * This going to get the act main script path for a specific act.
 */
func GetActScriptFilePath(actCallId string) string {
	dtdir := GetActDataDirPath(actCallId)

	return path.Join(dtdir, "script.sh")
}
