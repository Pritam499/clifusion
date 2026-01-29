// Command templates and macros
package cobra

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type TemplateManager struct {
	file string
}

func NewTemplateManager() *TemplateManager {
	dir := filepath.Join(os.Getenv("HOME"), ".cobra")
	os.MkdirAll(dir, 0755)
	return &TemplateManager{file: filepath.Join(dir, "templates.json")}
}

func (tm *TemplateManager) Load() (map[string]string, error) {
	data, err := os.ReadFile(tm.file)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	var templates map[string]string
	err = json.Unmarshal(data, &templates)
	return templates, err
}

func (tm *TemplateManager) Save(templates map[string]string) error {
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tm.file, data, 0644)
}

func (tm *TemplateManager) Add(name, command string) error {
	templates, err := tm.Load()
	if err != nil {
		return err
	}
	templates[name] = command
	return tm.Save(templates)
}

func (tm *TemplateManager) Get(name string) (string, error) {
	templates, err := tm.Load()
	if err != nil {
		return "", err
	}
	cmd, ok := templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}
	return cmd, nil
}

func (tm *TemplateManager) List() (map[string]string, error) {
	return tm.Load()
}

func (tm *TemplateManager) Delete(name string) error {
	templates, err := tm.Load()
	if err != nil {
		return err
	}
	delete(templates, name)
	return tm.Save(templates)
}

func CreateTemplateCommand() *Command {
	tm := NewTemplateManager()

	cmd := &Command{
		Use:   "template",
		Short: "Manage command templates",
	}

	saveCmd := &Command{
		Use:   "save <name> <command>",
		Short: "Save a command template",
		Args:  MinimumNArgs(2),
		RunE: func(cmd *Command, args []string) error {
			name := args[0]
			command := strings.Join(args[1:], " ")
			return tm.Add(name, command)
		},
	}

	runCmd := &Command{
		Use:   "run <name>",
		Short: "Run a command template",
		Args:  ExactArgs(1),
		RunE: func(cmd *Command, args []string) error {
			template, err := tm.Get(args[0])
			if err != nil {
				return err
			}
			// Execute the template
			parts := strings.Split(template, "&&")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				cmd := exec.Command("sh", "-c", part)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to run %s: %v", part, err)
				}
			}
			return nil
		},
	}

	listCmd := &Command{
		Use:   "list",
		Short: "List all templates",
		RunE: func(cmd *Command, args []string) error {
			templates, err := tm.List()
			if err != nil {
				return err
			}
			for name, command := range templates {
				fmt.Printf("%s: %s\n", name, command)
			}
			return nil
		},
	}

	deleteCmd := &Command{
		Use:   "delete <name>",
		Short: "Delete a template",
		Args:  ExactArgs(1),
		RunE: func(cmd *Command, args []string) error {
			return tm.Delete(args[0])
		},
	}

	cmd.AddCommand(saveCmd, runCmd, listCmd, deleteCmd)
	return cmd
}