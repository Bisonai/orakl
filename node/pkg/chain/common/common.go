package common

import (
	"fmt"
	"regexp"
	"strings"
)

// reference: https://github.com/umbracle/ethgo/blob/main/abi/abi.go
var (
	funcRegexpWithReturn    = regexp.MustCompile(`(\w*)\s*\((.*)\)(.*)\s*returns\s*\((.*)\)`)
	funcRegexpWithoutReturn = regexp.MustCompile(`(\w*)\s*\((.*)\)(.*)`)
)

func ParseMethodSignature(name string) (string, string, string, error) {
	name = strings.Replace(name, "\n", " ", -1)
	name = strings.Replace(name, "\t", " ", -1)

	name = strings.TrimPrefix(name, "function ")
	name = strings.TrimSpace(name)

	var funcName, inputArgs, outputArgs string

	if strings.Contains(name, "returns") {
		matches := funcRegexpWithReturn.FindAllStringSubmatch(name, -1)
		if len(matches) == 0 {
			return "", "", "", fmt.Errorf("no matches found")
		}
		funcName = strings.TrimSpace(matches[0][1])
		inputArgs = strings.TrimSpace(matches[0][2])
		outputArgs = strings.TrimSpace(matches[0][4])
	} else {
		matches := funcRegexpWithoutReturn.FindAllStringSubmatch(name, -1)
		if len(matches) == 0 {
			return "", "", "", fmt.Errorf("no matches found")
		}
		funcName = strings.TrimSpace(matches[0][1])
		inputArgs = strings.TrimSpace(matches[0][2])
	}

	return funcName, inputArgs, outputArgs, nil
}

func MakeAbiFuncAttribute(args string) string {
	splittedArgs := strings.Split(args, ",")
	if len(splittedArgs) == 0 || splittedArgs[0] == "" {
		return ""
	}

	var parts []string
	for _, arg := range splittedArgs {
		arg = strings.TrimSpace(arg)
		part := strings.Split(arg, " ")

		if len(part) < 2 {
			parts = append(parts, fmt.Sprintf(`{"type":"%s"}`, part[0]))
		} else {
			parts = append(parts, fmt.Sprintf(`{"type":"%s","name":"%s"}`, part[0], part[len(part)-1]))
		}
	}
	return strings.Join(parts, ",\n")
}
