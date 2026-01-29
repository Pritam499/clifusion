// Example of setting up pipeline commands
package cobra

import "fmt"

type Data struct {
	Content string
}

type JSONData struct {
	JSON string
}

func SetupPipelineExample(rootCmd *Command) {
	fetchCmd := &Command{
		Use:       "fetch",
		Short:     "Fetch data",
		OutputType: "data",
		PipelineRunE: func(cmd *Command, args []string, input interface{}) (interface{}, error) {
			return &Data{Content: "fetched data"}, nil
		},
	}

	transformCmd := &Command{
		Use:       "transform",
		Short:     "Transform data to JSON",
		InputType: "data",
		OutputType: "json",
		PipelineRunE: func(cmd *Command, args []string, input interface{}) (interface{}, error) {
			if data, ok := input.(*Data); ok {
				return &JSONData{JSON: `{"data": "` + data.Content + `"}`}, nil
			}
			return nil, fmt.Errorf("invalid input type")
		},
	}

	uploadCmd := &Command{
		Use:       "upload",
		Short:     "Upload JSON to S3",
		InputType: "json",
		PipelineRunE: func(cmd *Command, args []string, input interface{}) (interface{}, error) {
			if jsonData, ok := input.(*JSONData); ok {
				fmt.Println("Uploading to S3:", jsonData.JSON)
				return nil, nil
			}
			return nil, fmt.Errorf("invalid input type")
		},
	}

	rootCmd.AddCommand(fetchCmd, transformCmd, uploadCmd)
}