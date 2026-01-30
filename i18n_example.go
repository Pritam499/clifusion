// Example of setting up i18n aliases
package cobra

func SetupI18nExample(rootCmd *Command) {
	helpCmd := &Command{
		Use:   "help",
		Short: "Help about any command",
		Long:  `Help provides help for any command in the application.`,
		I18nAliases: map[string][]string{
			"es": {"ayuda"},
			"fr": {"aide"},
			"de": {"hilfe"},
		},
		Run: func(cmd *Command, args []string) {
			cmd.Println("Help command executed")
		},
	}
	rootCmd.AddCommand(helpCmd)
}