// Advanced completion helpers
package cobra

import (
	"os"
	"path/filepath"
	"strings"
)

// FileCompletion provides dynamic file completion based on current directory
func FileCompletion(cmd *Command, args []string, toComplete string) ([]Completion, ShellCompDirective) {
	files, err := filepath.Glob(toComplete + "*")
	if err != nil {
		return nil, ShellCompDirectiveDefault
	}

	var comps []Completion
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		desc := "file"
		if info.IsDir() {
			desc = "directory"
		}
		comps = append(comps, CompletionWithDesc(f, desc))
	}
	return comps, ShellCompDirectiveDefault
}

// GitContextCompletion provides git-aware completions
func GitContextCompletion(cmd *Command, args []string, toComplete string) ([]Completion, ShellCompDirective) {
	// Check if in git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		// Not in git repo, fallback to file completion
		return FileCompletion(cmd, args, toComplete)
	}

	// In git repo, provide git-specific completions
	var comps []Completion
	gitCmds := []string{"add", "commit", "push", "pull", "status", "log", "branch"}
	for _, gc := range gitCmds {
		if len(toComplete) == 0 || strings.HasPrefix(gc, toComplete) {
			comps = append(comps, CompletionWithDesc(gc, "git command"))
		}
	}
	return comps, ShellCompDirectiveNoFileComp
}

// DirectoryBasedCompletion switches completion based on current directory
func DirectoryBasedCompletion(cmd *Command, args []string, toComplete string) ([]Completion, ShellCompDirective) {
	cwd, err := os.Getwd()
	if err != nil {
		return FileCompletion(cmd, args, toComplete)
	}

	// Check for special directories
	if filepath.Base(cwd) == ".git" || filepath.Base(filepath.Dir(cwd)) == ".git" {
		return GitContextCompletion(cmd, args, toComplete)
	}

	// Default to file completion
	return FileCompletion(cmd, args, toComplete)
}