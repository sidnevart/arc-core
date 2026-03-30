package main

import (
	"fmt"
	"os"

	"agent-os/internal/ctxcli"
)

func main() {
	if err := ctxcli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
