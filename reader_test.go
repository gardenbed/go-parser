package parser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetModuleName(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedModule string
		expectedError  string
	}{
		{
			name:          "NoModFile",
			path:          "/opt",
			expectedError: "stat /opt/go.mod: no such file or directory",
		},
		{
			name:          "InvalidModule",
			path:          "./test/invalid_module",
			expectedError: "invalid go.mod file: no module name found",
		},
		{
			name:           "Success",
			path:           "./test/valid/lookup",
			expectedModule: "github.com/octocat/test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			module, err := getModuleName(tc.path)

			if tc.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedModule, module)
			} else {
				assert.Empty(t, module)
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestVisitPackages(t *testing.T) {
	successVisit := func(string, string) error {
		return nil
	}

	failVisit := func(string, string) error {
		return errors.New("generic error")
	}

	tests := []struct {
		name          string
		includeSubs   bool
		path          string
		visit         visitFunc
		expectedError string
	}{
		{
			name:          "PathNotExist",
			includeSubs:   false,
			path:          "./foo",
			visit:         successVisit,
			expectedError: "stat ./foo: no such file or directory",
		},
		{
			name:          "PathNotDirectory",
			includeSubs:   false,
			path:          "./test/valid/main.go",
			visit:         successVisit,
			expectedError: `"./test/valid/main.go" is not a directory`,
		},
		{
			name:          "Success_WithoutSubs",
			includeSubs:   false,
			path:          "./test/valid",
			visit:         successVisit,
			expectedError: "",
		},
		{
			name:          "Success_WithSubs",
			includeSubs:   true,
			path:          "./test/valid",
			visit:         successVisit,
			expectedError: "",
		},
		{
			name:          "VisitFails_FirstTime",
			includeSubs:   true,
			path:          "./test/valid",
			visit:         failVisit,
			expectedError: "generic error",
		},
		{
			name:        "VisitFails_SecondTime",
			includeSubs: true,
			path:        "./test/valid",
			visit: func(_, relPath string) error {
				if relPath == "." {
					return nil
				}
				return errors.New("generic error")
			},
			expectedError: "generic error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := visitPackages(tc.includeSubs, tc.path, tc.visit)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}
