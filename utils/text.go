package utils

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"
)

//############################################################
// Constants
//############################################################
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

//############################################################
// Exposed Functions
//############################################################

/**
 * This function going to convert a camel case text to a snake
 * case with all letters in uppercase.
 */
func CamelToSnakeUpperCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToUpper(snake)
}

/**
 * This function going to compile a go template text using
 * some variables.
 */
func CompileTemplate(text string, vars map[string]string) string {
	tpl, err := template.New("").Parse(text)

	if err != nil {
		FatalError("could not parse template", err)
	}

	var buff bytes.Buffer

	if err := tpl.Execute(&buff, vars); err != nil {
		FatalError("could not compile template", err)
	}

	return buff.String()
}
