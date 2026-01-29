// Visual Command Tree Builder
package cobra

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

//go:embed static/index.html
var indexHTML string

//go:embed static/app.js
var appJS string

type CommandNode struct {
	Name     string        `json:"name"`
	Use      string        `json:"use"`
	Short    string        `json:"short"`
	Flags    []FlagDef     `json:"flags,omitempty"`
	Children []CommandNode `json:"children,omitempty"`
}

type FlagDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

func CreateWebBuilderCommand() *Command {
	cmd := &Command{
		Use:   "builder",
		Short: "Launch visual command tree builder",
		RunE: func(cmd *Command, args []string) error {
			return startWebBuilder()
		},
	}
	return cmd
}

func startWebBuilder() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, indexHTML)
	})

	http.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, appJS)
	})

	http.HandleFunc("/generate", generateCodeHandler)

	fmt.Println("Starting web builder on http://localhost:8080")
	return http.ListenAndServe(":8080", nil)
}

func generateCodeHandler(w http.ResponseWriter, r *http.Request) {
	var tree CommandNode
	if err := json.NewDecoder(r.Body).Decode(&tree); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code := generateGoCode(tree)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, code)
}

func generateGoCode(node CommandNode) string {
	var code strings.Builder

	code.WriteString("package main\n\n")
	code.WriteString("import (\n")
	code.WriteString("\t\"github.com/spf13/cobra\"\n")
	code.WriteString(")\n\n")

	generateCommandCode(&code, node, "rootCmd")

	code.WriteString("func main() {\n")
	code.WriteString("\tif err := rootCmd.Execute(); err != nil {\n")
	code.WriteString("\t\tpanic(err)\n")
	code.WriteString("\t}\n")
	code.WriteString("}\n")

	return code.String()
}

func generateCommandCode(code *strings.Builder, node CommandNode, varName string) {
	code.WriteString(fmt.Sprintf("var %s = &cobra.Command{\n", varName))
	code.WriteString(fmt.Sprintf("\tUse:   \"%s\",\n", node.Use))
	code.WriteString(fmt.Sprintf("\tShort: \"%s\",\n", node.Short))
	code.WriteString("\tRun: func(cmd *cobra.Command, args []string) {\n")
	code.WriteString("\t\t// TODO: Implement command logic\n")
	code.WriteString("\t},\n")
	code.WriteString("}\n\n")

	// Add flags
	for _, flag := range node.Flags {
		switch flag.Type {
		case "string":
			code.WriteString(fmt.Sprintf("%s.Flags().String(\"%s\", \"\", \"%s\")\n", varName, flag.Name, flag.Description))
		case "bool":
			code.WriteString(fmt.Sprintf("%s.Flags().Bool(\"%s\", false, \"%s\")\n", varName, flag.Name, flag.Description))
		case "int":
			code.WriteString(fmt.Sprintf("%s.Flags().Int(\"%s\", 0, \"%s\")\n", varName, flag.Name, flag.Description))
		}
	}
	if len(node.Flags) > 0 {
		code.WriteString("\n")
	}

	// Add subcommands
	for i, child := range node.Children {
		childVar := fmt.Sprintf("%sCmd%d", varName, i)
		generateCommandCode(code, child, childVar)
		code.WriteString(fmt.Sprintf("%s.AddCommand(%s)\n", varName, childVar))
	}
}