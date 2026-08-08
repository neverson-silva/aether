package main

import (
	"fmt"
	"os"

	"aether/internal/cli"
)

func main() {
	commands := cli.Commands()
	if len(os.Args) < 2 {
		printUsage(commands)
		os.Exit(1)
	}
	name := os.Args[1]
	if name == "-h" || name == "--help" || name == "help" {
		printUsage(commands)
		os.Exit(0)
	}
	for _, c := range commands {
		if c.Name == name {
			if err := c.Run(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "erro:", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	fmt.Fprintf(os.Stderr, "comando desconhecido: %s\n\n", name)
	printUsage(commands)
	os.Exit(1)
}

func printUsage(commands []cli.Command) {
	fmt.Println("Aether — Plataforma Self-Hosted PaaS")
	fmt.Println()
	fmt.Println("Uso: aether <comando> [argumentos]")
	fmt.Println()
	fmt.Println("Comandos:")
	for _, c := range commands {
		fmt.Printf("  %-20s %s\n", c.Name, c.Usage)
	}
}
