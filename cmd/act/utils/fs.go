/**
 * This file expose functions to handle file system operations.
 */

package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

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
 * This function going to resolve a file path from a base dir path
 * if the file path is relative. Otherwise if file path is absolute
 * we going to return it instead.
 */
func ResolvePath(baseDir string, targetPath string) string {
	var thePath string

	if filepath.IsAbs(targetPath) {
		thePath = targetPath
	} else {
		thePath = path.Join(baseDir, targetPath)
	}

	return thePath
}
