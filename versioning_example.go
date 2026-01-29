// Example of command versioning and migrations
package cobra

import (
	"fmt"
)

func SetupVersioningExample(rootCmd *Command) {
	// Define a command with version
	cmd := &Command{
		Use:     "example",
		Short:   "Example command with versioning",
		Version: "1.0.0",
		Run: func(cmd *Command, args []string) {
			fmt.Println("Running example command v1.0.0")
		},
	}
	cmd.Flags().String("old-flag", "", "An old flag")

	// Register migration
	GlobalVersionManager.AddMigration("1.0.0", "2.0.0", func(from, to string, cmd *Command) error {
		fmt.Printf("Migrating %s from %s to %s\n", cmd.CommandPath(), from, to)
		// Example: rename flag
		cmd.Flags().String("new-flag", "", "A new flag")
		cmd.Flags().MarkDeprecated("old-flag", "use --new-flag instead")
		return nil
	})

	rootCmd.AddCommand(cmd)
}