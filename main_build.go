package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Benchkram/bob/bob"
	"github.com/Benchkram/bob/bob/build"
	"github.com/Benchkram/errz"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the project",
	Args:  cobra.MinimumNArgs(0),
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		dummy, err := strconv.ParseBool(cmd.Flag("dummy").Value.String())
		errz.Fatal(err)

		taskname := build.DefaultBuildTask
		if len(args) > 0 {
			taskname = args[0]
		}

		runBuild(dummy, taskname)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tasks, err := getTasks()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return tasks, cobra.ShellCompDirectiveDefault
	},
}

var buildListCmd = &cobra.Command{
	Use:   "ls",
	Short: "ls",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		runBuildList()
	},
}

func runBuild(dummy bool, taskname string) {
	if dummy {
		wd, err := os.Getwd()
		errz.Fatal(err)
		err = build.CreateDummyBobfile(wd, false)
		errz.Fatal(err)
		os.Exit(0)
	}

	b, err := bob.Bob()
	errz.Fatal(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		<-stop
		cancel()
	}()

	err = b.Build(ctx, taskname)
	switch err {
	case bob.ErrNoRebuildRequired:
	case context.Canceled:
	default:
		errz.Log(err)
	}
}

func runBuildList() {
	b, err := bob.Bob()
	errz.Log(err)

	err = b.List()
	errz.Log(err)
}

func getTasks() ([]string, error) {
	b, err := bob.Bob()
	if err != nil {
		return nil, err
	}
	return b.GetList()
}
