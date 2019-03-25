package main

import (
	"fmt"
	"os"

	"github.com/edgestore/edgestore/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const ShortDescription = "master"
const LongDescription = "Edgestore: Distributed Data Store (Master)"

func commandRoot() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   ShortDescription,
		Short: ShortDescription,
		Long:  LongDescription,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(2)
		},
	}

	viper.SetEnvPrefix("edgestore_master")
	viper.AutomaticEnv()

	rootCmd.AddCommand(commandServe())
	rootCmd.AddCommand(version.NewCommand(LongDescription))

	return rootCmd
}

func main() {
	if err := commandRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}
