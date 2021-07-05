package utils

import (
	"fmt"
)

/**
 * This function going to convert a vars map to a list of dotenv vars format
 * of "key=val" which we going to pass to command environment.
 */
func VarsMapToEnvVars(varsMap map[string]string) []string {
	var envVars []string

	for name, value := range varsMap {
		envVars = append(envVars, fmt.Sprintf("%s=%s", CamelToSnakeUpperCase(name), value))
	}

	return envVars
}
