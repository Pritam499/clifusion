// Built-in testing framework
package cobra

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type TestCase struct {
	Name        string            `yaml:"name"`
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	ExpectExit  int               `yaml:"expect_exit,omitempty"`
	ExpectOutput string           `yaml:"expect_output,omitempty"`
	ExpectError string            `yaml:"expect_error,omitempty"`
	Mocks       map[string]string `yaml:"mocks,omitempty"`
}

type TestSuite struct {
	Tests []TestCase `yaml:"tests"`
}

func CreateTestCommand() *Command {
	cmd := &Command{
		Use:   "test <file>",
		Short: "Run tests from YAML file",
		Args:  ExactArgs(1),
		RunE: func(cmd *Command, args []string) error {
			return runTests(args[0])
		},
	}
	return cmd
}

func runTests(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var suite TestSuite
	err = yaml.Unmarshal(data, &suite)
	if err != nil {
		return err
	}

	passed := 0
	failed := 0

	for _, test := range suite.Tests {
		fmt.Printf("Running test: %s\n", test.Name)
		err := runTestCase(test)
		if err != nil {
			fmt.Printf("  FAILED: %v\n", err)
			failed++
		} else {
			fmt.Printf("  PASSED\n")
			passed++
		}
	}

	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)
	return nil
}

func runTestCase(test TestCase) error {
	// Set up environment
	env := os.Environ()
	for k, v := range test.Env {
		env = append(env, k+"="+v)
	}

	// Set up mocks
	if len(test.Mocks) > 0 {
		mockDir, err := os.MkdirTemp("", "cobra_test_mocks")
		if err != nil {
			return err
		}
		defer os.RemoveAll(mockDir)

		for mockCmd, mockScript := range test.Mocks {
			mockPath := filepath.Join(mockDir, mockCmd)
			err := os.WriteFile(mockPath, []byte(mockScript), 0755)
			if err != nil {
				return err
			}
		}

		// Prepend mock dir to PATH
		env = append(env, "PATH="+mockDir+":"+os.Getenv("PATH"))
	}

	// Prepare command
	var cmd *exec.Cmd
	if len(test.Args) > 0 {
		cmd = exec.Command(test.Command, test.Args...)
	} else {
		// If no args, assume command is shell command
		cmd = exec.Command("sh", "-c", test.Command)
	}

	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return fmt.Errorf("failed to run command: %v", err)
		}
	}

	// Check expectations
	if test.ExpectExit != 0 && exitCode != test.ExpectExit {
		return fmt.Errorf("expected exit code %d, got %d", test.ExpectExit, exitCode)
	}

	outStr := stdout.String()
	if test.ExpectOutput != "" && !strings.Contains(outStr, test.ExpectOutput) {
		return fmt.Errorf("expected output to contain '%s', got '%s'", test.ExpectOutput, outStr)
	}

	errStr := stderr.String()
	if test.ExpectError != "" && !strings.Contains(errStr, test.ExpectError) {
		return fmt.Errorf("expected error to contain '%s', got '%s'", test.ExpectError, errStr)
	}

	return nil
}