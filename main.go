package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func main() {
    var rootCmd = &cobra.Command{Use: "mycli"}

    var helloCmd = &cobra.Command{
        Use:   "hello",
        Short: "Prints hello message",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Hello, World!")
        },
    }

    rootCmd.AddCommand(helloCmd)
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
