package common

import (
	"fmt"
	"testing"
)

func TestParseMethodSignature(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedName   string
		expectedInput  string
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "Test case 1",
			input:          "function foo()",
			expectedName:   "foo",
			expectedInput:  "",
			expectedOutput: "",
			expectedError:  nil,
		},
		{
			name:           "Test case 2",
			input:          "function bar(arg1 int, arg2 string) returns (bool)",
			expectedName:   "bar",
			expectedInput:  "arg1 int, arg2 string",
			expectedOutput: "bool",
			expectedError:  nil,
		},
		{
			name:           "Test case 3",
			input:          "function baz(arg1 int, arg2 string) returns (bool, string)",
			expectedName:   "baz",
			expectedInput:  "arg1 int, arg2 string",
			expectedOutput: "bool, string",
			expectedError:  nil,
		},
		{
			name:           "Test case 4",
			input:          "",
			expectedName:   "",
			expectedInput:  "",
			expectedOutput: "",
			expectedError:  fmt.Errorf("empty name"),
		},
	}

	for _, test := range tests {
		funcName, inputArgs, outputArgs, err := ParseMethodSignature(test.input)
		if err != nil && err.Error() != test.expectedError.Error() {
			t.Errorf("Test case %s: Expected error '%v', but got '%v'", test.name, test.expectedError, err)
		}
		if funcName != test.expectedName {
			t.Errorf("Test case %s: Expected function name '%s', but got '%s'", test.name, test.expectedName, funcName)
		}
		if inputArgs != test.expectedInput {
			t.Errorf("Test case %s: Expected input arguments '%s', but got '%s'", test.name, test.expectedInput, inputArgs)
		}
		if outputArgs != test.expectedOutput {
			t.Errorf("Test case %s: Expected output arguments '%s', but got '%s'", test.name, test.expectedOutput, outputArgs)
		}
	}
}

func TestMakeAbiFuncAttribute(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		expected string
	}{
		{
			name:     "Test case 1",
			args:     "",
			expected: "",
		},
		{
			name:     "Test case 2",
			args:     "int",
			expected: `{"type":"int"}`,
		},
		{
			name:     "Test case 3",
			args:     "string",
			expected: `{"type":"string"}`,
		},
		{
			name: "Test case 4",
			args: "int, string",
			expected: `{"type":"int"},
{"type":"string"}`,
		},
		{
			name: "Test case 5",
			args: "int arg1, string arg2",
			expected: `{"type":"int","name":"arg1"},
{"type":"string","name":"arg2"}`,
		},
		{
			name: "Test case 6",
			args: "address[] memory addresses, uint256[] memory amounts",
			expected: `{"type":"address[]","name":"addresses"},
{"type":"uint256[]","name":"amounts"}`,
		},
	}

	for _, test := range tests {
		result := MakeAbiFuncAttribute(test.args)
		if result != test.expected {
			t.Errorf("Test case %s: Expected '%s', but got '%s'", test.name, test.expected, result)
		}
	}
}
