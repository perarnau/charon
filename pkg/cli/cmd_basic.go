package cli

import (
	"fmt"
	"os"
)

func (c *CLI) executeHelp(args []string) error {
	fmt.Println("📖 Available Commands:")
	fmt.Println("===================")

	for _, cmd := range c.commands {
		fmt.Printf("  %-12s - %s\n", cmd.Name, cmd.Description)
	}

	fmt.Println("\nUsage: <command> [arguments...]")
	return nil
}

func (c *CLI) executeExit(args []string) error {
	// Ensure terminal is properly restored
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset all attributes
	fmt.Println("👋 Goodbye!")
	os.Exit(0)
	return nil
}
