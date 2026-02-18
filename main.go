package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "telegram-nats-bridge",
		Short: "Bridge between Telegram Bot API and NATS",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the bridge",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("hello world")
		},
	}

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
