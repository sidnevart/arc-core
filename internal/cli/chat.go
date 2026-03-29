package cli

import (
	"flag"
	"fmt"
	"io"

	"agent-os/internal/app"
	"agent-os/internal/project"
)

func runChat(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("chat requires a subcommand")
	}
	switch args[0] {
	case "start":
		return runChatStart(args[1:])
	case "send":
		return runChatSend(args[1:])
	case "list":
		return runChatList(args[1:])
	case "show":
		return runChatShow(args[1:])
	default:
		return fmt.Errorf("unknown chat subcommand %q", args[0])
	}
}

func runChatStart(args []string) error {
	fs := flag.NewFlagSet("chat start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	providerFlag := fs.String("provider", "", "provider override")
	modeFlag := fs.String("mode", "", "mode override")
	modelFlag := fs.String("model", "", "model override")
	dryRunFlag := fs.Bool("dry-run", false, "skip provider execution")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: arc chat start [--path DIR] [--provider NAME] [--mode MODE] [--model MODEL] [--dry-run] [--json] <prompt>")
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	proj, err := project.Load(root)
	if err != nil {
		return err
	}
	providerName := proj.Config.DefaultProvider
	if *providerFlag != "" {
		providerName = *providerFlag
	}
	modeName := proj.Mode.Mode
	if *modeFlag != "" {
		modeName = *modeFlag
	}
	svc := app.NewService()
	detail, err := svc.StartChat(root, providerName, modeName, *modelFlag, fs.Args()[0], *dryRunFlag, "explain", false, nil)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(detail)
	}
	fmt.Printf("chat %s started with %s\n", detail.Session.ID, detail.Session.Provider)
	return nil
}

func runChatSend(args []string) error {
	fs := flag.NewFlagSet("chat send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	sessionFlag := fs.String("session", "", "chat session id")
	modelFlag := fs.String("model", "", "model override")
	dryRunFlag := fs.Bool("dry-run", false, "skip provider execution")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: arc chat send [--path DIR] [--session ID] [--model MODEL] [--dry-run] [--json] <prompt>")
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	svc := app.NewService()
	detail, err := svc.SendChat(root, *sessionFlag, *modelFlag, fs.Args()[0], *dryRunFlag, "explain", false, nil)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(detail)
	}
	fmt.Printf("chat %s updated\n", detail.Session.ID)
	return nil
}

func runChatList(args []string) error {
	fs := flag.NewFlagSet("chat list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	limitFlag := fs.Int("limit", 20, "max sessions")
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
	sessions, err := app.NewService().ListChats(root, *limitFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(sessions)
	}
	for _, session := range sessions {
		fmt.Printf("%s %s %s\n", session.ID, session.Provider, session.Status)
	}
	return nil
}

func runChatShow(args []string) error {
	fs := flag.NewFlagSet("chat show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	sessionFlag := fs.String("session", "", "chat session id")
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
	detail, err := app.NewService().ChatDetail(root, *sessionFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(detail)
	}
	fmt.Printf("chat: %s\nprovider: %s\nstatus: %s\n", detail.Session.ID, detail.Session.Provider, detail.Session.Status)
	for _, message := range detail.Messages {
		fmt.Printf("[%s] %s\n%s\n\n", message.Role, message.CreatedAt, message.Content)
	}
	return nil
}
