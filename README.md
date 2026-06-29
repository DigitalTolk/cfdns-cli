# cfdns-cli

[![CI](https://github.com/DigitalTolk/cfdns-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/DigitalTolk/cfdns-cli/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/DigitalTolk/cfdns-cli/badge.svg?branch=main)](https://coveralls.io/github/DigitalTolk/cfdns-cli?branch=main)

An interactive terminal browser for Cloudflare DNS zones and records.

Its reason to exist: when you `terraform import` a Cloudflare DNS record you
need the record's **ID**, and there's no good way to get it from the dashboard.
`cfdns-cli` lets you browse zones, fuzzy-search records, and copy the record ID
(or zone ID) straight from a TUI.

## Install

Homebrew (macOS / Linux) — installs the `cfdns` command:

```sh
brew install DigitalTolk/tools/cfdns-cli
```

Pre-built binaries for Linux, macOS, and Windows (amd64 / arm64) are attached to
each [GitHub release](https://github.com/DigitalTolk/cfdns-cli/releases).

Or build from source:

```sh
go build -o cfdns .
```

## Authentication

Cloudflare does not offer an OAuth browser login for third-party CLIs, so this
tool uses a **scoped API token**, which it stores in your OS keychain (macOS
Keychain / Windows Credential Manager / Secret Service on Linux) — never in a
plaintext file.

On first run you'll be prompted to paste a token. Create one at
<https://dash.cloudflare.com/profile/api-tokens> with these permissions:

- **Zone → Zone → Read**
- **Zone → DNS → Read**

The token is verified before it's saved, so a bad token never gets stored.

## Usage

```sh
cfdns           # browse
cfdns -login    # replace the stored token
cfdns -logout   # delete the stored token
```

### Keys

| Screen   | Key            | Action                                       |
| -------- | -------------- | -------------------------------------------- |
| any list | `/`            | text filter (matches type, name, content)    |
| any list | `up`/`down`    | move                                         |
| records  | `t`            | cycle record-type filter (all → A → MX → …)  |
| zones    | `enter`        | open the zone's records                      |
| records  | `enter`        | open record details                          |
| records  | `esc`          | back to zones (or clear text filter)         |
| any      | `r`            | refresh from the API                         |
| detail   | `c`            | copy **record ID** to clipboard              |
| detail   | `z`            | copy **zone ID** to clipboard                |
| detail   | `n`            | copy record name to clipboard                |
| detail   | `esc`          | back to records                              |
| any      | `q` / `ctrl+c` | quit                                         |

Two ways to filter records: press `/` and type to fuzzy-search (the query matches
type, name, and content), or press `t` to cycle a strict filter through only the
record types present in the zone — the active type is shown in the title bar. The
two compose: set a type with `t`, then `/`-search within it.

The detail screen shows the record ID and zone ID prominently — copy the ID,
then drop it into your `terraform import`:

```sh
terraform import cloudflare_record.www <zone_id>/<record_id>
```

## Stack

- [cloudflare-go/v4](https://github.com/cloudflare/cloudflare-go) — official SDK
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) + Bubbles + Lip Gloss — TUI
- [go-keyring](https://github.com/zalando/go-keyring) — OS keychain storage
