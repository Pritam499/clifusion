// Distributed command execution
package cobra

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

type ExecutionResult struct {
	Host    string
	Success bool
	Output  string
	Error   string
}

type DistributedExecutor struct {
	Hosts     []string
	User      string
	KeyPath   string
	Password  string
	Parallel  bool
}

func CreateDistributedCommand() *Command {
	cmd := &Command{
		Use:   "distribute <command>",
		Short: "Execute command on multiple remote hosts",
		Args:  MinimumNArgs(1),
		RunE: func(cmd *Command, args []string) error {
			command := strings.Join(args, " ")

			hosts, _ := cmd.Flags().GetStringSlice("hosts")
			user, _ := cmd.Flags().GetString("user")
			keyPath, _ := cmd.Flags().GetString("key")
			password, _ := cmd.Flags().GetString("password")
			parallel, _ := cmd.Flags().GetBool("parallel")

			if len(hosts) == 0 {
				return fmt.Errorf("no hosts specified")
			}

			executor := &DistributedExecutor{
				Hosts:    hosts,
				User:     user,
				KeyPath:  keyPath,
				Password: password,
				Parallel: parallel,
			}

			results, err := executor.Execute(command)
			if err != nil {
				return err
			}

			// Aggregate and display results
			return displayResults(results)
		},
	}

	cmd.Flags().StringSliceP("hosts", "H", []string{}, "Target hosts (comma-separated)")
	cmd.Flags().StringP("user", "u", "root", "SSH username")
	cmd.Flags().StringP("key", "k", "", "SSH private key path")
	cmd.Flags().StringP("password", "p", "", "SSH password")
	cmd.Flags().BoolP("parallel", "P", true, "Execute in parallel")

	return cmd
}

func (de *DistributedExecutor) Execute(command string) ([]ExecutionResult, error) {
	var results []ExecutionResult
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	if de.Parallel {
		for _, host := range de.Hosts {
			wg.Add(1)
			go func(h string) {
				defer wg.Done()
				result := de.executeOnHost(h, command)
				mu.Lock()
				results = append(results, result)
				if !result.Success {
					errs = append(errs, fmt.Errorf("host %s failed: %s", h, result.Error))
				}
				mu.Unlock()
			}(host)
		}
		wg.Wait()
	} else {
		for _, host := range de.Hosts {
			result := de.executeOnHost(host, command)
			results = append(results, result)
			if !result.Success {
				errs = append(errs, fmt.Errorf("host %s failed: %s", host, result.Error))
			}
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("execution errors: %v", errs)
	}

	return results, nil
}

func (de *DistributedExecutor) executeOnHost(host, command string) ExecutionResult {
	result := ExecutionResult{Host: host}

	config := &ssh.ClientConfig{
		User: de.User,
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if de.KeyPath != "" {
		key, err := os.ReadFile(de.KeyPath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("read key: %v", err)
			return result
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("parse key: %v", err)
			return result
		}
		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	if de.Password != "" {
		config.Auth = append(config.Auth, ssh.Password(de.Password))
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("dial: %v", err)
		return result
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("new session: %v", err)
		return result
	}
	defer session.Close()

	var stdoutBuf, stderrBuf strings.Builder
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(command)
	result.Output = stdoutBuf.String()
	result.Error = stderrBuf.String()

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("run: %v\n%s", err, result.Error)
	} else {
		result.Success = true
	}

	return result
}

func displayResults(results []ExecutionResult) error {
	fmt.Println("Execution Results:")
	fmt.Println("==================")

	successCount := 0
	for _, result := range results {
		status := "✓"
		if !result.Success {
			status = "✗"
		} else {
			successCount++
		}

		fmt.Printf("[%s] %s\n", status, result.Host)
		if result.Output != "" {
			fmt.Printf("Output: %s\n", result.Output)
		}
		if result.Error != "" {
			fmt.Printf("Error: %s\n", result.Error)
		}
		fmt.Println()
	}

	fmt.Printf("Summary: %d/%d hosts succeeded\n", successCount, len(results))

	if successCount != len(results) {
		return fmt.Errorf("some executions failed")
	}

	return nil
}