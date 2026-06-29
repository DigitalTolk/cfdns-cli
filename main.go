// cfdns-cli is an interactive terminal browser for Cloudflare DNS zones and
// records. Its purpose is making it painless to find the record ID you need
// for `terraform import`.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DigitalTolk/cfdns-cli/internal/auth"
	"github.com/DigitalTolk/cfdns-cli/internal/cf"
	"github.com/DigitalTolk/cfdns-cli/internal/tui"
)

// Build information, injected at release time via -ldflags by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		logout      = flag.Bool("logout", false, "remove the stored API token and exit")
		login       = flag.Bool("login", false, "prompt for a new API token even if one is stored")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("cfdns version %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if *logout {
		if err := auth.Delete(); err != nil {
			fatal(err)
		}
		fmt.Println("Stored token removed.")
		return
	}

	token, err := loadOrPromptToken(*login)
	if err != nil {
		fatal(err)
	}

	p := tea.NewProgram(tui.New(cf.New(token)), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fatal(err)
	}
}

// loadOrPromptToken returns a usable token, prompting and storing one the first
// time (or when force is set). A freshly entered token is verified before it is
// persisted so we never save a broken one.
func loadOrPromptToken(force bool) (string, error) {
	if !force {
		tok, err := auth.Load()
		if err == nil {
			return tok, nil
		}
		if !errors.Is(err, auth.ErrNoToken) {
			return "", err
		}
	}

	tok, err := auth.Prompt()
	if err != nil {
		return "", err
	}

	fmt.Fprintln(os.Stderr, "Verifying token...")
	if err := cf.New(tok).Verify(context.Background()); err != nil {
		return "", err
	}
	if err := auth.Store(tok); err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr, "Token saved to keychain.")
	return tok, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
