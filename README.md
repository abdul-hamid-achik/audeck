# audeck

A terminal UI for managing macOS audio devices. Switch defaults, adjust volume, toggle mute — all from your terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and CoreAudio.

## Features

- **Output & Input devices** — tab between output and input device lists
- **Default device switching** — set the system default with `enter`
- **Volume control** — coarse (5%) and fine (1%) adjustments
- **Mute toggle** — per-device mute with `m`
- **Real-time updates** — event-driven via CoreAudio listeners, no polling
- **Hot-plug support** — devices appear/disappear as you connect/disconnect them
- **Per-channel volume fallback** — works with USB audio interfaces that expose per-channel volume (VolumeScalar) instead of VirtualMainVolume
- **Mouse support** — scroll wheel navigation
- **Catppuccin Mocha** theme

## Install

### Homebrew

```sh
brew install abdul-hamid-achik/tap/audeck
```

### From source

Requires Go 1.24+ and macOS.

```sh
git clone https://github.com/abdul-hamid-achik/audeck.git
cd audeck
go build -o audeck ./cmd/audeck
./audeck
```

## Usage

### TUI Mode

```sh
audeck [flags]
```

| Flag | Description |
|------|-------------|
| `--version` | Print version and exit |
| `--debug` | Enable debug logging to `/tmp/audeck.log` |
| `-j, --json` | Output in JSON format (for CLI commands) |
| `--config` | Override config file path |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `AUDECK_DEBUG=1` | Enable debug logging |
| `AUDECK_CONFIG=path` | Override config file path |

### CLI Commands

Audeck also provides a full-featured CLI for scripting and automation:

```sh
# List all audio devices
audeck list
audeck list --type output    # Only output devices
audeck list --type input     # Only input devices
audeck list --json           # JSON output

# Search for devices by name
audeck search "speaker"
audeck search "bluetooth" --json

# Get default device
audeck default get                 # Default output
audeck default get input           # Default input
audeck default get --json          # JSON output

# Set default device
audeck default set 76              # By device ID
audeck default set "Speakers"      # By device name
audeck default set --type input 83 # Set input device

# Volume control
audeck volume get                  # Get default device volume
audeck volume get 76               # Get specific device volume
audeck volume set 50               # Set volume to 50%
audeck volume up                   # Increase by 5%
audeck volume down 10              # Decrease by 10%
audeck volume up --device 76       # Control specific device

# Mute control
audeck mute get                    # Get mute state
audeck mute set on                 # Mute
audeck mute set off                # Unmute
audeck mute toggle                 # Toggle mute

# System info
audeck info                        # Audio system information
audeck info --json                 # JSON output

# Shell completions
audeck completion bash > /etc/bash_completion.d/audeck
audeck completion zsh > /usr/local/share/zsh/site-functions/_audeck
audeck completion fish > ~/.config/fish/completions/audeck.fish
```

### Keybindings (TUI)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `tab` / `1` / `2` | Switch between Output and Input tabs |
| `enter` | Set selected device as system default |
| `h` / `l` | Volume down/up (5% steps) |
| `+` / `-` | Fine volume adjust (1% steps) |
| `m` | Toggle mute |
| `g` / `G` | Jump to first/last device |
| `q` | Quit |
| Scroll wheel | Navigate up/down |

## Configuration

Audeck saves your preferred default devices to `~/.config/audeck/config.json`. On launch, it restores your preferred output and input defaults if the devices are connected.

## Architecture

```
cmd/audeck/          CLI entry point
internal/
  coreaudio/         CoreAudio C bindings (CGo)
  audio/             Device manager, event system
  config/            Persistent config (~/.config/audeck/)
  tui/               Bubble Tea model, view, styles
```

Three layers with clean separation:

1. **CoreAudio** — low-level property queries and listeners via CGo
2. **Audio Manager** — thread-safe device state, event subscription, volume/mute with per-channel fallback
3. **TUI** — Bubble Tea model consuming events from the manager

## Requirements

- macOS (uses CoreAudio, AudioToolbox, CoreFoundation frameworks)
- Go 1.24+

## License

MIT
