package cli

import (
	"errors"
	"fmt"
)

func Run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "doctor":
		return runDoctor(args[1:])
	case "init":
		return runInit(args[1:])
	case "workspace":
		return runWorkspace(args[1:])
	case "chat":
		return runChat(args[1:])
	case "mode":
		return runMode(args[1:])
	case "index":
		return runIndex(args[1:])
	case "task":
		return runTask(args[1:])
	case "budget":
		return runBudget(args[1:])
	case "hook":
		return runHook(args[1:])
	case "preset":
		return runPreset(args[1:])
	case "docs":
		return runDocs(args[1:])
	case "memory":
		return runMemory(args[1:])
	case "questions":
		return runQuestions(args[1:])
	case "run":
		return runRun(args[1:])
	case "learn":
		return runLearn(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage() {
	fmt.Println("arc - Agent Runtime CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  arc doctor")
	fmt.Println("  arc init [--path DIR] [--provider codex,claude] [--mode study|work|hero]")
	fmt.Println("  arc workspace summary [--path DIR] [--json]")
	fmt.Println("  arc workspace repair [--path DIR] [--json]")
	fmt.Println("  arc workspace verify [--path DIR] [--profile ID | --verifier ID] [--json]")
	fmt.Println("  arc chat start [--path DIR] [--provider NAME] [--mode MODE] [--model MODEL] [--dry-run] [--json] <prompt>")
	fmt.Println("  arc chat send [--path DIR] [--session ID] [--model MODEL] [--dry-run] [--json] <prompt>")
	fmt.Println("  arc chat list [--path DIR] [--limit N] [--json]")
	fmt.Println("  arc chat show [--path DIR] [--session ID] [--json]")
	fmt.Println("  arc mode set <study|work|hero> [--path DIR]")
	fmt.Println("  arc mode show [--path DIR] [--json]")
	fmt.Println("  arc index build [--path DIR]")
	fmt.Println("  arc index refresh [--path DIR]")
	fmt.Println("  arc task plan [--path DIR] [--mode MODE] [--provider NAME] [--budget-mode MODE] <task>")
	fmt.Println("  arc task run [--path DIR] [--mode MODE] [--provider NAME] [--budget-mode MODE] [--budget-override-file FILE] [--dry-run] [--run-checks] [--approve-risky] [--provider-timeout DURATION] <task>")
	fmt.Println("  arc budget show [--path DIR] [--json]")
	fmt.Println("  arc budget override set [--path DIR] [--mode MODE] [--low-limit-state STATE] [--prefer-local true|false] [--block-premium-high-risk true|false] [--block-premium-required true|false] [--require-approval-for-premium-high-risk true|false] [--require-approval-for-premium-required true|false] [--note TEXT] [--json]")
	fmt.Println("  arc budget override clear [--path DIR] [--json]")
	fmt.Println("  arc budget session show --file FILE [--json]")
	fmt.Println("  arc budget session write --file FILE [--mode MODE] [--low-limit-state STATE] [--prefer-local true|false] [--block-premium-high-risk true|false] [--block-premium-required true|false] [--require-approval-for-premium-high-risk true|false] [--require-approval-for-premium-required true|false] [--note TEXT] [--json]")
	fmt.Println("  arc budget session clear --file FILE [--json]")
	fmt.Println("  arc task verify [--path DIR] [--run-id ID] [--run-checks]")
	fmt.Println("  arc task review [--path DIR] [--run-id ID]")
	fmt.Println("  arc hook memory add [--path DIR] --scope SCOPE [--kind KIND] [--tags a,b] [--json] <summary>")
	fmt.Println("  arc preset list [--root DIR] [--json]")
	fmt.Println("  arc preset draft init --path DIR --id ID --name NAME --summary TEXT --goal TEXT [--type domain|infrastructure|session_overlay] [--target-agent codex|claude|arc] [--providers a,b] [--version X.Y.Z] [--budget-profile MODE] [--autonomy low|medium|high] [--json]")
	fmt.Println("  arc preset draft update --path DIR --id ID [--name NAME] [--summary TEXT] [--goal TEXT] [--type TYPE] [--target-agent AGENT] [--providers a,b] [--version X.Y.Z] [--budget-profile MODE] [--autonomy low|medium|high] [--non-goals a,b] [--inputs a,b] [--outputs a,b] [--workflow a,b] [--quality-gates a,b] [--status draft|tested|published|installed|deprecated|archived] [--json]")
	fmt.Println("  arc preset draft interview start --path DIR --id ID [--mode quick|deep|import-refine] [--json]")
	fmt.Println("  arc preset draft interview answer --path DIR --session ID [--json] <answer>")
	fmt.Println("  arc preset draft interview remediate --path DIR [--json] <session-id>")
	fmt.Println("  arc preset draft interview show --path DIR [--json] <session-id>")
	fmt.Println("  arc preset draft simulate --path DIR [--json] <preset-id>")
	fmt.Println("  arc preset draft mark-tested --path DIR [--json] <preset-id>")
	fmt.Println("  arc preset draft export --path DIR [--json] <preset-id>")
	fmt.Println("  arc preset draft publish --path DIR [--json] <preset-id>")
	fmt.Println("  arc preset draft catalog-sync --path DIR --root DIR [--json] <preset-id>")
	fmt.Println("  arc preset draft install --path DIR [--force] [--json] <preset-id>")
	fmt.Println("  arc preset draft show [--path DIR] [--json] <preset-id>")
	fmt.Println("  arc preset draft validate [--path DIR] [--json] <preset-id>")
	fmt.Println("  arc preset draft list [--path DIR] [--json]")
	fmt.Println("  arc preset validate [--root DIR] [--json] <preset-id>")
	fmt.Println("  arc preset preview [--root DIR] [--path DIR] [--json] <preset-id>")
	fmt.Println("  arc preset install [--root DIR] [--path DIR] [--force] [--json] <preset-id>")
	fmt.Println("  arc preset rollback [--path DIR] [--json] <install-id>")
	fmt.Println("  arc docs generate [--path DIR] [--run-id ID] [--apply]")
	fmt.Println("  arc memory status [--path DIR] [--json]")
	fmt.Println("  arc memory compact [--path DIR]")
	fmt.Println("  arc questions show [--path DIR]")
	fmt.Println("  arc run list [--path DIR] [--limit N] [--json]")
	fmt.Println("  arc run status [--path DIR] [--run-id ID]")
	fmt.Println("  arc run resume [--path DIR] [--run-id ID] [--model MODEL] [--dry-run] [prompt]")
	fmt.Println("  arc learn <topic>")
	fmt.Println("  arc learn quiz [--path DIR]")
	fmt.Println("  arc learn prove <claim>")
}

func requireNoArgs(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected arguments: %v", args)
	}
	return nil
}

func requireMode(mode string) error {
	switch mode {
	case "study", "work", "hero":
		return nil
	default:
		return errors.New("mode must be one of: study, work, hero")
	}
}
