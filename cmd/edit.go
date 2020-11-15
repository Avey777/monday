package main

import (
	"fmt"
	"os/exec"

	"github.com/eko/monday/internal/runtime"
	"github.com/eko/monday/pkg/config"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "This command allows you to open the configuration file in your default editor",
	Long: `For more information about the configuration, see the "example" directory available
in the source code repository at https://github.com/eko/monday.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check for multiple configuration file
		files := config.FindMultipleConfigFiles()

		// Check for single configuration file
		err := config.CheckConfigFileExists()
		if err != nil {
			fmt.Printf("❌  %v\n", err)
			return
		}

		if len(files) == 0 {
			files = []string{config.Filepath}
		}

		editorArgs := append(runtime.EditorArgs, files...)
		editorCommand := exec.Command(runtime.EditorCommand, editorArgs...)

		if err := editorCommand.Start(); err != nil {
			fmt.Printf("❌  Cannot run the '%s' command to edit config file: %v\n", runtime.EditorCommand, err)
			return
		}
	},
}
