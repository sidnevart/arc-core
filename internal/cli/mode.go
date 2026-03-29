package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/app"
	"agent-os/internal/mode"
	"agent-os/internal/project"
)

func runMode(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("mode requires a subcommand")
	}

	switch args[0] {
	case "set":
		return runModeSet(args[1:])
	case "show":
		return runModeShow(args[1:])
	default:
		return fmt.Errorf("unknown mode subcommand %q", args[0])
	}
}

func runModeSet(args []string) error {
	modeName, path, err := parseModeSetArgs(args)
	if err != nil {
		return err
	}
	if modeName == "" {
		return fmt.Errorf("usage: arc mode set <study|work|hero> [--path DIR]")
	}
	if err := requireMode(modeName); err != nil {
		return err
	}

	root, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := project.RequireProject(root); err != nil {
		return err
	}

	if err := project.WriteMode(root, modeName, mode.ByName(modeName).Autonomy); err != nil {
		return err
	}
	if err := project.AppendEvent(root, project.Event{
		Timestamp: time.Now().UTC(),
		Command:   "mode set",
		Status:    "ok",
		Details: map[string]string{
			"mode": modeName,
			"path": root,
		},
	}); err != nil {
		return err
	}

	fmt.Printf("active mode set to %s\n", modeName)
	return nil
}

func runModeShow(args []string) error {
	fs := flag.NewFlagSet("mode show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}

	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	proj, err := project.Load(root)
	if err != nil {
		return err
	}
	def := mode.ByName(proj.Mode.Mode)
	if *jsonFlag {
		return writeJSON(map[string]any{
			"mode":     proj.Mode.Mode,
			"autonomy": def.Autonomy,
			"goal":     def.Goal,
			"workspace": app.WorkspaceSummary{
				Root: proj.Root,
				Name: proj.Config.Name,
			},
		})
	}
	fmt.Printf("mode: %s\n", proj.Mode.Mode)
	fmt.Printf("autonomy: %s\n", def.Autonomy)
	fmt.Printf("goal: %s\n", def.Goal)
	return nil
}

func parseModeSetArgs(args []string) (string, string, error) {
	path := "."
	mode := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--path":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("missing value for --path")
			}
			path = args[i+1]
			i++
		case strings.HasPrefix(arg, "--path="):
			path = strings.TrimPrefix(arg, "--path=")
		case strings.HasPrefix(arg, "-"):
			return "", "", fmt.Errorf("unknown flag %q", arg)
		case mode == "":
			mode = arg
		default:
			return "", "", fmt.Errorf("unexpected argument %q", arg)
		}
	}

	return mode, path, nil
}
