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

```
audeck [flags]
```

| Flag | Description |
|------|-------------|
| `--version` | Print version and exit |
| `--debug` | Enable debug logging to `/tmp/audeck.log` |

### Keybindings

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
