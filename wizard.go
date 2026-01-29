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
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	}
}

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateInputFlag:
			if msg.String() == "enter" {
				value := m.input.Value()
				m.flags[m.currentFlag] = value
				m.builtArgs = append(m.builtArgs, "--"+m.currentFlag, value)
				m.state = stateSelectFlags
				return m, nil
			} else if msg.String() == "esc" {
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
							m.input.SetValue("")
							m.input.Focus()
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
		b.WriteString("Select a subcommand (or enter for root):\n\n")
		for i, cmd := range m.subCommands {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, cmd))
		}
		b.WriteString("\n(enter to select, q to quit)\n")
	case stateSelectFlags:
		b.WriteString(fmt.Sprintf("Flags for %s:\n\n", m.currentCmd.Use))
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
		b.WriteString(fmt.Sprintf("Enter value for --%s (%s):\n\n", m.currentFlag, m.currentCmd.Flags().Lookup(m.currentFlag).Usage))
		b.WriteString(m.input.View())
		b.WriteString("\n\n(enter to set, esc to cancel)\n")
	case stateDone:
		b.WriteString("Built command: ")
		b.WriteString(strings.Join(append([]string{m.rootCmd.Use}, m.builtArgs...), " "))
		b.WriteString("\n\nPress enter to confirm, q to quit\n")
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