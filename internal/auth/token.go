// Package auth handles persistence and entry of the Cloudflare API token.
//
// The token is stored in the operating system keychain (macOS Keychain,
// Windows Credential Manager, or the Secret Service / kwallet on Linux) via
// github.com/zalando/go-keyring, so it never lands in a plaintext config file.
package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const (
	keyringService = "cfdns-cli"
	keyringUser    = "api-token"
)

// ErrNoToken is returned by Load when no token is stored yet.
var ErrNoToken = errors.New("no API token stored")

// Load returns the stored API token, or ErrNoToken if none is set.
func Load() (string, error) {
	tok, err := keyring.Get(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNoToken
	}
	if err != nil {
		return "", fmt.Errorf("reading token from keychain: %w", err)
	}
	return tok, nil
}

// Store saves the API token to the OS keychain.
func Store(token string) error {
	if err := keyring.Set(keyringService, keyringUser, token); err != nil {
		return fmt.Errorf("writing token to keychain: %w", err)
	}
	return nil
}

// Delete removes the stored API token. It is not an error if none exists.
func Delete() error {
	err := keyring.Delete(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deleting token from keychain: %w", err)
	}
	return nil
}

// Prompt asks the user to paste an API token on the terminal without echoing
// it. It returns the trimmed token.
func Prompt() (string, error) {
	fmt.Fprintln(os.Stderr, "Create a scoped token at:")
	fmt.Fprintln(os.Stderr, "  https://dash.cloudflare.com/profile/api-tokens")
	fmt.Fprintln(os.Stderr, "Permissions needed: Zone > Zone (Read) and Zone > DNS (Read).")
	fmt.Fprint(os.Stderr, "\nPaste Cloudflare API token: ")

	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after the hidden input
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	tok := strings.TrimSpace(string(raw))
	if tok == "" {
		return "", errors.New("empty token")
	}
	return tok, nil
}
