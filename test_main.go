package main

import (
    "fmt"
    "os"
    
    "github.com/Pritam499/clifusion"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "testcli",
        Short: "Test CLI for CliFusion features",
    }
    
    // Add a server command
    serverCmd := &cobra.Command{
        Use:   "server",
        Short: "Start the server",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("✅ Server started successfully!")
        },
    }
    
    // Add help command with i18n
    helpCmd := &cobra.Command{
        Use:   "help",
        Short: "Show help",
        I18nAliases: map[string][]string{
            "es": {"ayuda"},
            "fr": {"aide"},
        },
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("✅ Help command executed!")
        },
    }
    
    rootCmd.AddCommand(serverCmd, helpCmd)
    
    // Enable smart suggestions
    rootCmd.SuggestionsMinimumDistance = 1
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
