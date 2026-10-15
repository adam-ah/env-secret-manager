package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

func interactive() error {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return errors.New("interactive workspace requires a terminal; use help, list, or exec for automation")
	}
	if os.Getenv("TERM") == "dumb" {
		return errors.New("interactive workspace needs a terminal with cursor support")
	}
	return nil
}

func openMenu() error {
	if err := interactive(); err != nil {
		fmt.Print(helpText)
		return nil
	}
	return launchWorkspace()
}

func editSecrets() error {
	return openMenu()
}

func launchWorkspace() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newWorkspace(cfg), tea.WithAltScreen()).Run()
	return err
}

func showInstall(args []string) error {
	shell := filepath.Base(os.Getenv("SHELL"))
	if shell != "bash" && shell != "zsh" {
		shell = "zsh"
	}
	if len(args) > 1 {
		return errors.New("usage: secretenv install [bash|zsh]")
	}
	if len(args) == 1 {
		shell = args[0]
	}
	if shell != "bash" && shell != "zsh" {
		return errors.New("supported shells: bash, zsh")
	}
	fmt.Printf("Add this to ~/.%src, then open a new shell:\n\n", shell)
	fmt.Print(shellHook(shell))
	fmt.Print("\nKeep secretenv on PATH. Project secrets load for the current folder and clear\nwhen you leave it. Global is never auto-injected. No files were changed.\nUse a shell without this hook or inherited secrets for AI sessions.\n")
	return nil
}

func shellHook(shell string) string {
	hook := `_secretenv_hook() {
  local out
  out="$(secretenv _hook)" || return
  eval "$out"
}
`
	if shell == "zsh" {
		return hook + "autoload -Uz add-zsh-hook\nadd-zsh-hook -d chpwd _secretenv_hook\nadd-zsh-hook chpwd _secretenv_hook\n_secretenv_hook\n"
	}
	return hook + `if [[ $(declare -p PROMPT_COMMAND 2>/dev/null) == "declare -a "* ]]; then
  if [[ " ${PROMPT_COMMAND[*]} " != *" _secretenv_hook "* ]]; then
    PROMPT_COMMAND=(_secretenv_hook "${PROMPT_COMMAND[@]}")
  fi
else
  case ";${PROMPT_COMMAND-};" in
    *";_secretenv_hook;"*) ;;
    *) PROMPT_COMMAND="_secretenv_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
  esac
fi
_secretenv_hook
`
}
