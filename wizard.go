// Copyright 2013-2023 The Cobra Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cobra

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

type wizardState int

const (
	stateSelectCommand wizardState = iota
	stateSelectFlags
	stateInputFlag
	stateDone
)

type wizardModel struct {
	rootCmd      *Command
	currentCmd   *Command
	subCommands  []string
	cursor       int
	state        wizardState
	flags        map[string]string
	flagCursor   int
	flagList     []string
	currentFlag  string
	input        textinput.Model
	builtArgs    []string
	errorMsg     string
	flagType     string
	flagExample  string
}

func getFlagExampleAndType(flagName string, flag *flag.Flag) (string, string) {
	// Determine type
	flagType := "string"
	if strings.Contains(flag.Value.String(), ".") {
		flagType = "float"
	} else if _, err := strconv.Atoi(flag.Value.String()); err == nil {
		flagType = "int"
	} else if flag.Value.String() == "true" || flag.Value.String() == "false" {
		flagType = "bool"
	}

	// Provide examples based on name or type
	switch strings.ToLower(flagName) {
	case "port":
		return "8080", "int"
	case "host", "hostname":
		return "localhost", "string"
	case "timeout":
		return "30s", "string"
	case "verbose", "debug":
		return "true", "bool"
	case "output", "file":
		return "output.txt", "string"
	case "count", "number":
		return "5", "int"
	default:
		switch flagType {
		case "int":
			return "42", "int"
		case "float":
			return "3.14", "float"
		case "bool":
			return "true", "bool"
		default:
			return "example_value", "string"
		}
	}
}

func initialWizardModel(root *Command, initialArgs []string) wizardModel {
	subCmds := make([]string, 0, len(root.commands))
	for _, cmd := range root.commands {
		if cmd.IsAvailableCommand() {
			subCmds = append(subCmds, cmd.Name())
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Enter value"

	return wizardModel{
		rootCmd:     root,
		currentCmd:  root,
		subCommands: subCmds,
		cursor:      0,
		state:       stateSelectCommand,
		flags:       make(map[string]string),
		input:       ti,
		builtArgs:   initialArgs,
		errorMsg:    "",
		flagType:    "",
		flagExample: "",
	}
}

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *wizardModel) validateFlagValue(value string) (bool, string) {
	switch m.flagType {
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return false, "Please enter a valid integer"
		}
	case "float":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return false, "Please enter a valid number"
		}
	case "bool":
		if value != "true" && value != "false" {
			return false, "Please enter 'true' or 'false'"
		}
	}
	return true, ""
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateInputFlag:
			if msg.String() == "enter" {
				value := m.input.Value()
				// Validate input
				if valid, err := m.validateFlagValue(value); valid {
					m.flags[m.currentFlag] = value
					m.builtArgs = append(m.builtArgs, "--"+m.currentFlag, value)
					m.errorMsg = ""
					m.state = stateSelectFlags
				} else {
					m.errorMsg = err
				}
				return m, nil
			} else if msg.String() == "esc" {
				m.errorMsg = ""
				m.state = stateSelectFlags
				return m, nil
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		default:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				switch m.state {
				case stateSelectCommand:
					if len(m.subCommands) == 0 || m.cursor < 0 || m.cursor >= len(m.subCommands) {
						// No subcommand or select none
						m.state = stateSelectFlags
						m.flagList = []string{"Done"}
						m.currentCmd.Flags().VisitAll(func(f *flag.Flag) {
							m.flagList = append([]string{f.Name}, m.flagList...)
						})
						m.flagCursor = 0
					} else {
						selected := m.subCommands[m.cursor]
						for _, cmd := range m.currentCmd.commands {
							if cmd.Name() == selected {
								m.currentCmd = cmd
								m.builtArgs = append(m.builtArgs, selected)
								m.state = stateSelectFlags
								m.flagList = []string{"Done"}
								cmd.Flags().VisitAll(func(f *flag.Flag) {
									m.flagList = append([]string{f.Name}, m.flagList...)
								})
								m.flagCursor = 0
								break
							}
						}
					}
				case stateSelectFlags:
					if m.flagCursor >= 0 && m.flagCursor < len(m.flagList) {
						selected := m.flagList[m.flagCursor]
						if selected == "Done" {
							m.state = stateDone
						} else {
							m.currentFlag = selected
							f := m.currentCmd.Flags().Lookup(selected)
							m.flagExample, m.flagType = getFlagExampleAndType(selected, f)
							m.input.SetValue("")
							m.input.Focus()
							m.errorMsg = ""
							m.state = stateInputFlag
						}
					}
				case stateDone:
					return m, tea.Quit
				}
			case "up":
				if m.state == stateSelectCommand && m.cursor > 0 {
					m.cursor--
				} else if m.state == stateSelectFlags && m.flagCursor > 0 {
					m.flagCursor--
				}
			case "down":
				if m.state == stateSelectCommand && m.cursor < len(m.subCommands)-1 {
					m.cursor++
				} else if m.state == stateSelectFlags && m.flagCursor < len(m.flagList)-1 {
					m.flagCursor++
				}
			}
		}
	}
	return m, cmd
}

func (m wizardModel) View() string {
	var b strings.Builder

	switch m.state {
	case stateSelectCommand:
		b.WriteString("Welcome to the Interactive Command Builder Wizard!\n")
		b.WriteString("Build your command step by step with visual guidance.\n\n")
		b.WriteString("Select a subcommand (or press enter for root command):\n\n")
		for i, cmd := range m.subCommands {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, cmd))
		}
		b.WriteString("\n(arrow keys to navigate, enter to select, q to quit)\n")
	case stateSelectFlags:
		b.WriteString(fmt.Sprintf("Select flags for '%s' command:\n", m.currentCmd.Use))
		b.WriteString("Use arrow keys to navigate, enter to select, q to quit\n\n")
		for i, item := range m.flagList {
			cursor := " "
			if m.flagCursor == i {
				cursor = ">"
			}
			if item == "Done" {
				b.WriteString(fmt.Sprintf("%s %s\n", cursor, item))
			} else {
				f := m.currentCmd.Flags().Lookup(item)
				desc := f.Usage
				b.WriteString(fmt.Sprintf("%s --%s: %s\n", cursor, item, desc))
			}
		}
		b.WriteString("\n(enter to set flag or done, q to quit)\n")
	case stateInputFlag:
		f := m.currentCmd.Flags().Lookup(m.currentFlag)
		b.WriteString(fmt.Sprintf("Enter value for --%s\n", m.currentFlag))
		b.WriteString(fmt.Sprintf("Description: %s\n", f.Usage))
		b.WriteString(fmt.Sprintf("Type: %s\n", m.flagType))
		b.WriteString(fmt.Sprintf("Example: %s\n\n", m.flagExample))
		b.WriteString(m.input.View())
		if m.errorMsg != "" {
			b.WriteString(fmt.Sprintf("\nError: %s\n", m.errorMsg))
		}
		b.WriteString("\n\n(enter to set, esc to cancel)\n")
	case stateDone:
		b.WriteString("Command built successfully!\n\n")
		b.WriteString("Final command: ")
		b.WriteString(strings.Join(append([]string{m.rootCmd.Use}, m.builtArgs...), " "))
		b.WriteString("\n\nThis command will be executed when you confirm.")
		b.WriteString("\n\nPress enter to confirm and run, q to quit without running\n")
	}

	return b.String()
}

func launchWizard(c *Command, initialArgs []string) []string {
	m := initialWizardModel(c, initialArgs)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil
	}
	if final, ok := finalModel.(wizardModel); ok && final.state == stateDone {
		return final.builtArgs
	}
	return nil
}