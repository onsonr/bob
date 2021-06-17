package main

import (
	"fmt"
	"os"

	"github.com/Benchkram/bob/bob"
	"github.com/Benchkram/errz"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect Tasks of a Bobfile",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "List environment for a task",
	Args:  cobra.ExactArgs(1),
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		taskname := args[0]
		runEnv(taskname)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tasks, err := getTasks()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return tasks, cobra.ShellCompDirectiveDefault
	},
}

func runEnv(taskname string) {
	b, err := bob.Bob()
	errz.Log(err)

	bobfile, err := b.Aggregate()
	errz.Log(err)

	task, ok := bobfile.Tasks[taskname]
	if !ok {
		fmt.Printf("%s\n", aurora.Red("Task does not exists"))
		os.Exit(1)
	}

	for _, env := range task.Env() {
		println(env)
	}
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "List exports for a task",
	Args:  cobra.ExactArgs(1),
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		taskname := args[0]
		runExport(taskname)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tasks, err := getTasks()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return tasks, cobra.ShellCompDirectiveDefault
	},
}

func runExport(taskname string) {
	b, err := bob.Bob()
	errz.Log(err)

	bobfile, err := b.Aggregate()
	errz.Log(err)

	task, ok := bobfile.Tasks[taskname]
	if !ok {
		fmt.Printf("%s\n", aurora.Red("Task does not exists"))
		os.Exit(1)
	}

	for exportname, export := range task.GetExports() {
		fmt.Printf("%s (%s)\n", exportname, export)
	}
}
