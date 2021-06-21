package utils

import "fmt"

/**
 * This function converts a vars map to an array of env vars.
 *
 * @param varsMap - Map of variables we want to convert to env vars list.
 */
func VarsMapToEnvVars(varsMap map[string]interface{}) []string {
	var envVars []string

	/**
	 * @TODO : We should allow map of maps here and convert something
	 * like this:
	 *
	 * ```json
	 * {
	 *   "foo": {
	 *      "bar": "value" 
	 *   }
	 * }
	 * ```
	 * 
	 * to something like this ["FOO_BAR=value"]. Maybe we have a package
	 * to do this.
	 */
	for name, value := range varsMap {
		envVar := fmt.Sprintf("%s=%s", CamelToSnakeUpperCase(name), value)
		envVars = append(envVars, envVar)
	}

	return envVars
}
