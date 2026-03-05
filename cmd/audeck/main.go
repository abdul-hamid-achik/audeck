package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/abdulachik/audeck/internal/audio"
	"github.com/abdulachik/audeck/internal/config"
	"github.com/abdulachik/audeck/internal/tui"
)

var (
	version    = "dev"
	debugMode  bool
	jsonOutput bool
	configPath string
)

func init() {
	// Environment variable overrides
	if os.Getenv("AUDECK_DEBUG") == "1" {
		debugMode = true
	}
	if path := os.Getenv("AUDECK_CONFIG"); path != "" {
		configPath = path
	}
}

func main() {
	if err := fang.Execute(context.Background(), rootCmd(), fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audeck",
		Short: "Terminal UI for macOS audio devices",
		Long: lipgloss.NewStyle().Bold(true).Render("audeck") + " - Audio Device Manager for macOS\n\n" +
			"A terminal UI for managing macOS audio devices. Switch defaults, adjust volume, toggle mute — all from your terminal.\n\n" +
			"ENVIRONMENT VARIABLES\n\n" +
			"  AUDECK_DEBUG=1       Enable debug logging\n" +
			"  AUDECK_CONFIG=path   Override config file path\n",
		RunE: runTUI,
	}

	cmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging to /tmp/audeck.log")
	cmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format (for CLI commands)")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "Override config file path")

	cmd.AddCommand(
		listCmd(),
		defaultCmd(),
		volumeCmd(),
		muteCmd(),
		infoCmd(),
		searchCmd(),
		profileCmd(),
		configCmd(),
	)

	return cmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	manager, err := audio.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize audio manager: %w", err)
	}
	defer manager.Close()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.DefaultOutputDevice != "" {
		_ = manager.ApplyPreferredDevice(cfg.DefaultOutputDevice, audio.ScopeOutput)
	}
	if cfg.DefaultInputDevice != "" {
		_ = manager.ApplyPreferredDevice(cfg.DefaultInputDevice, audio.ScopeInput)
	}

	model := tui.NewModel(manager)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}

	return nil
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all audio devices",
		Long:  "List all available audio output and input devices with their current state.",
		RunE:  runList,
	}
	cmd.Flags().StringP("type", "t", "all", "Filter by type: all, output, input")
	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	manager, err := audio.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize audio manager: %w", err)
	}
	defer manager.Close()

	filter, _ := cmd.Flags().GetString("type")
	var devices []audio.Device
	switch filter {
	case "output":
		devices = manager.OutputDevices()
	case "input":
		devices = manager.InputDevices()
	default:
		devices = manager.Devices()
	}

	if jsonOutput {
		return outputJSON(devices)
	}
	return outputDeviceList(devices, filter)
}

func defaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default",
		Short: "Manage default audio devices",
		Long:  "Get or set the default output or input audio device.",
	}
	cmd.AddCommand(defaultGetCmd(), defaultSetCmd())
	return cmd
}

func defaultGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [output|input]",
		Short: "Get current default device",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			scope := "output"
			if len(args) > 0 {
				scope = args[0]
			}

			var device audio.Device
			var found bool
			if scope == "input" {
				device, found = manager.DefaultInputDevice()
			} else {
				device, found = manager.DefaultDevice()
			}

			if !found {
				return errors.New("no default device found")
			}

			if jsonOutput {
				return outputJSON(device)
			}

			fmt.Printf("Default %s device: %s\n", scope, device.Name)
			if device.HasVolume {
				fmt.Printf("  Volume: %d%%\n", device.VolumePercent())
			}
			if device.HasMute {
				fmt.Printf("  Muted: %v\n", device.Muted)
			}
			return nil
		},
	}
}

func defaultSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <device-id>",
		Short: "Set default device",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			deviceType, _ := cmd.Flags().GetString("type")
			deviceArg := args[0]

			deviceID, err := strconv.ParseUint(deviceArg, 10, 32)
			if err == nil {
				if deviceType == "input" {
					if err := manager.SetDefaultInputDevice(uint32(deviceID)); err != nil {
						return err
					}
				} else {
					if err := manager.SetDefaultDevice(uint32(deviceID)); err != nil {
						return err
					}
				}
				fmt.Printf("Default %s device set to ID %d\n", deviceType, deviceID)
				return nil
			}

			devices := manager.Devices()
			for _, d := range devices {
				if d.Name == deviceArg {
					if deviceType == "input" && !d.IsInput {
						return fmt.Errorf("device %s is not an input device", deviceArg)
					}
					if deviceType != "input" && !d.IsOutput {
						return fmt.Errorf("device %s is not an output device", deviceArg)
					}
					if deviceType == "input" {
						if err := manager.SetDefaultInputDevice(d.ID); err != nil {
							return err
						}
					} else {
						if err := manager.SetDefaultDevice(d.ID); err != nil {
							return err
						}
					}
					fmt.Printf("Default %s device set to: %s\n", deviceType, d.Name)
					return nil
				}
			}
			return fmt.Errorf("device not found: %s", deviceArg)
		},
	}
	cmd.Flags().StringP("type", "t", "output", "Device type: output or input")
	return cmd
}

func volumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Control device volume",
		Long:  "Get or set the volume of an audio device.",
	}
	cmd.AddCommand(volumeGetCmd(), volumeSetCmd(), volumeUpCmd(), volumeDownCmd())
	return cmd
}

func volumeGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [device-id]",
		Short: "Get device volume",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			var deviceID uint32
			if len(args) > 0 {
				id, err := strconv.ParseUint(args[0], 10, 32)
				if err != nil {
					return err
				}
				deviceID = uint32(id)
			} else {
				d, found := manager.DefaultDevice()
				if !found {
					return errors.New("no default device found")
				}
				deviceID = d.ID
			}

			device, found := manager.DeviceByID(deviceID)
			if !found {
				return fmt.Errorf("device %d not found", deviceID)
			}

			if !device.HasVolume {
				return errors.New("device does not support volume control")
			}

			if jsonOutput {
				return outputJSON(map[string]interface{}{
					"device_id": device.ID,
					"device":    device.Name,
					"volume":    device.Volume,
					"percent":   device.VolumePercent(),
					"muted":     device.Muted,
				})
			}

			fmt.Printf("Device: %s\n", device.Name)
			fmt.Printf("Volume: %d%%\n", device.VolumePercent())
			if device.Muted {
				fmt.Println("Status: Muted")
			}
			return nil
		},
	}
}

func volumeSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <percent>",
		Short: "Set device volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			percent, err := strconv.ParseFloat(args[0], 32)
			if err != nil {
				return err
			}
			if percent < 0 || percent > 100 {
				return errors.New("percentage must be between 0 and 100")
			}

			deviceID, _ := cmd.Flags().GetUint32("device")
			if deviceID == 0 {
				d, found := manager.DefaultDevice()
				if !found {
					return errors.New("no default device found")
				}
				deviceID = d.ID
			}

			device, found := manager.DeviceByID(deviceID)
			if !found {
				return fmt.Errorf("device %d not found", deviceID)
			}

			if !device.HasVolume {
				return errors.New("device does not support volume control")
			}

			if err := manager.SetVolume(deviceID, float32(percent/100.0)); err != nil {
				return err
			}

			fmt.Printf("Volume set to %d%% on %s\n", int(percent), device.Name)
			return nil
		},
	}
	cmd.Flags().Uint32P("device", "d", 0, "Device ID (uses default if not specified)")
	return cmd
}

func volumeUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [delta]",
		Short: "Increase volume",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return adjustVolume(cmd, args, true)
		},
	}
	cmd.Flags().Uint32P("device", "d", 0, "Device ID")
	return cmd
}

func volumeDownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down [delta]",
		Short: "Decrease volume",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return adjustVolume(cmd, args, false)
		},
	}
	cmd.Flags().Uint32P("device", "d", 0, "Device ID")
	return cmd
}

func adjustVolume(cmd *cobra.Command, args []string, up bool) error {
	manager, err := audio.NewManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	delta := float32(5.0)
	if len(args) > 0 {
		d, err := strconv.ParseFloat(args[0], 32)
		if err != nil {
			return err
		}
		delta = float32(d)
	}
	if !up {
		delta = -delta
	}

	deviceID, _ := cmd.Flags().GetUint32("device")
	if deviceID == 0 {
		d, found := manager.DefaultDevice()
		if !found {
			return errors.New("no default device found")
		}
		deviceID = d.ID
	}

	device, found := manager.DeviceByID(deviceID)
	if !found {
		return fmt.Errorf("device %d not found", deviceID)
	}

	if !device.HasVolume {
		return errors.New("device does not support volume control")
	}

	if err := manager.AdjustVolume(deviceID, delta/100.0); err != nil {
		return err
	}

	direction := "increased"
	if !up {
		direction = "decreased"
	}
	fmt.Printf("Volume %s by %.0f%% on %s\n", direction, delta, device.Name)
	return nil
}

func muteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mute",
		Short: "Control device mute state",
	}
	cmd.AddCommand(muteGetCmd(), muteSetCmd(), muteToggleCmd())
	return cmd
}

func muteGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [device-id]",
		Short: "Get mute state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			var deviceID uint32
			if len(args) > 0 {
				id, err := strconv.ParseUint(args[0], 10, 32)
				if err != nil {
					return err
				}
				deviceID = uint32(id)
			} else {
				d, found := manager.DefaultDevice()
				if !found {
					return errors.New("no default device found")
				}
				deviceID = d.ID
			}

			device, found := manager.DeviceByID(deviceID)
			if !found {
				return fmt.Errorf("device %d not found", deviceID)
			}

			if !device.HasMute {
				return errors.New("device does not support mute control")
			}

			if jsonOutput {
				return outputJSON(map[string]interface{}{
					"device_id": device.ID,
					"device":    device.Name,
					"muted":     device.Muted,
				})
			}

			status := "unmuted"
			if device.Muted {
				status = "muted"
			}
			fmt.Printf("Device: %s\nStatus: %s\n", device.Name, status)
			return nil
		},
	}
}

func muteSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <on|off>",
		Short: "Set mute state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			muted := args[0] == "on" || args[0] == "true"
			deviceID, _ := cmd.Flags().GetUint32("device")
			if deviceID == 0 {
				d, found := manager.DefaultDevice()
				if !found {
					return errors.New("no default device found")
				}
				deviceID = d.ID
			}

			device, found := manager.DeviceByID(deviceID)
			if !found {
				return fmt.Errorf("device %d not found", deviceID)
			}

			if !device.HasMute {
				return errors.New("device does not support mute control")
			}

			if err := manager.SetMute(deviceID, muted); err != nil {
				return err
			}

			status := "muted"
			if !muted {
				status = "unmuted"
			}
			fmt.Printf("Device %s is now %s\n", device.Name, status)
			return nil
		},
	}
	cmd.Flags().Uint32P("device", "d", 0, "Device ID")
	return cmd
}

func muteToggleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toggle",
		Short: "Toggle mute state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			deviceID, _ := cmd.Flags().GetUint32("device")
			if deviceID == 0 {
				d, found := manager.DefaultDevice()
				if !found {
					return errors.New("no default device found")
				}
				deviceID = d.ID
			}

			device, found := manager.DeviceByID(deviceID)
			if !found {
				return fmt.Errorf("device %d not found", deviceID)
			}

			if !device.HasMute {
				return errors.New("device does not support mute control")
			}

			if err := manager.ToggleMute(deviceID); err != nil {
				return err
			}

			newStatus := "muted"
			if !device.Muted {
				newStatus = "unmuted"
			}
			fmt.Printf("Device %s is now %s\n", device.Name, newStatus)
			return nil
		},
	}
	cmd.Flags().Uint32P("device", "d", 0, "Device ID")
	return cmd
}

func infoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show system audio information",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			outputDevices := manager.OutputDevices()
			inputDevices := manager.InputDevices()
			defaultOutput, _ := manager.DefaultDevice()
			defaultInput, _ := manager.DefaultInputDevice()

			if jsonOutput {
				info := map[string]interface{}{
					"output_devices": len(outputDevices),
					"input_devices":  len(inputDevices),
					"default_output": defaultOutput,
					"default_input":  defaultInput,
					"config":         cfg,
				}
				return outputJSON(info)
			}

			fmt.Println("Audio System Information")
			fmt.Println("========================")
			fmt.Printf("Output devices: %d\n", len(outputDevices))
			fmt.Printf("Input devices:  %d\n", len(inputDevices))
			fmt.Println()

			if defaultOutput.Name != "" {
				fmt.Printf("Default output: %s (ID: %d)\n", defaultOutput.Name, defaultOutput.ID)
			} else {
				fmt.Println("Default output: (none)")
			}

			if defaultInput.Name != "" {
				fmt.Printf("Default input:  %s (ID: %d)\n", defaultInput.Name, defaultInput.ID)
			} else {
				fmt.Println("Default input:  (none)")
			}

			fmt.Println()
			fmt.Println("Configuration:")
			cfgPath, _ := config.Path()
			fmt.Printf("  Config file: %s\n", cfgPath)
			if cfg.DefaultOutputDevice != "" {
				fmt.Printf("  Preferred output: %s\n", cfg.DefaultOutputDevice)
			}
			if cfg.DefaultInputDevice != "" {
				fmt.Printf("  Preferred input:  %s\n", cfg.DefaultInputDevice)
			}
			return nil
		},
	}
}

func searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <pattern>",
		Short: "Search for devices by name",
		Long:  "Search for audio devices matching a pattern (case-insensitive).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			pattern := strings.ToLower(args[0])
			devices := manager.Devices()
			var matches []audio.Device

			for _, d := range devices {
				if strings.Contains(strings.ToLower(d.Name), pattern) {
					matches = append(matches, d)
				}
			}

			if jsonOutput {
				return outputJSON(matches)
			}

			if len(matches) == 0 {
				fmt.Printf("No devices found matching '%s'\n", pattern)
				return nil
			}

			fmt.Printf("Found %d device(s) matching '%s':\n\n", len(matches), pattern)
			return outputDeviceList(matches, "search")
		},
	}
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
	return cmd
}

func profileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage audio profiles",
		Long:  "Save and load audio device configurations as profiles.",
	}
	cmd.AddCommand(
		profileSaveCmd(),
		profileLoadCmd(),
		profileListCmd(),
		profileDeleteCmd(),
		profileActiveCmd(),
	)
	return cmd
}

func profileSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save current device state as a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			profile := config.Profile{
				Name: args[0],
			}

			// Save current default devices
			if out, ok := manager.DefaultDevice(); ok {
				profile.DefaultOutput = out.UID
			}
			if in, ok := manager.DefaultInputDevice(); ok {
				profile.DefaultInput = in.UID
			}

			// Save current volume levels
			if out, ok := manager.DefaultDevice(); ok && out.HasVolume {
				profile.OutputVolume = out.Volume
			}
			if out, ok := manager.DefaultDevice(); ok && out.HasMute {
				profile.Muted = out.Muted
			}

			cfg.AddProfile(profile)
			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Profile '%s' saved\n", args[0])
			return nil
		},
	}
	return cmd
}

func profileLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load <name>",
		Short: "Load a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := audio.NewManager()
			if err != nil {
				return err
			}
			defer manager.Close()

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			profile, found := cfg.GetProfile(args[0])
			if !found {
				return fmt.Errorf("profile '%s' not found", args[0])
			}

			// Apply default devices
			if profile.DefaultOutput != "" {
				_ = manager.ApplyPreferredDevice(profile.DefaultOutput, audio.ScopeOutput)
			}
			if profile.DefaultInput != "" {
				_ = manager.ApplyPreferredDevice(profile.DefaultInput, audio.ScopeInput)
			}

			// Apply volume settings
			if profile.OutputVolume > 0 {
				if dev, ok := manager.DefaultDevice(); ok && dev.HasVolume {
					_ = manager.SetVolume(dev.ID, profile.OutputVolume)
				}
			}
			if dev, ok := manager.DefaultDevice(); ok && dev.HasMute {
				_ = manager.SetMute(dev.ID, profile.Muted)
			}

			// Set active profile
			cfg.ActiveProfile = profile.Name
			_ = cfg.Save()

			fmt.Printf("Profile '%s' loaded\n", profile.Name)
			return nil
		},
	}
	return cmd
}

func profileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			profiles := cfg.ListProfiles()
			if len(profiles) == 0 {
				fmt.Println("No profiles saved")
				return nil
			}

			fmt.Println("Saved profiles:")
			for _, name := range profiles {
				marker := " "
				if name == cfg.ActiveProfile {
					marker = "*"
				}
				fmt.Printf("  %s %s\n", marker, name)
			}
			fmt.Println("\n* = active profile")
			return nil
		},
	}
}

func profileDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if !cfg.DeleteProfile(args[0]) {
				return fmt.Errorf("profile '%s' not found", args[0])
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Profile '%s' deleted\n", args[0])
			return nil
		},
	}
}

func profileActiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active",
		Short: "Show or set active profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				if cfg.ActiveProfile != "" {
					fmt.Printf("Active profile: %s\n", cfg.ActiveProfile)
				} else {
					fmt.Println("No active profile")
				}
				return nil
			}

			_, found := cfg.GetProfile(args[0])
			if !found {
				return fmt.Errorf("profile '%s' not found", args[0])
			}

			cfg.ActiveProfile = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Active profile set to '%s'\n", args[0])
			return nil
		},
	}
	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "View and edit audeck configuration including keybindings.",
	}
	cmd.AddCommand(
		configShowCmd(),
		configKeybindingsCmd(),
		configThemeCmd(),
	)
	return cmd
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			path, _ := config.Path()
			fmt.Println("Configuration:")
			fmt.Printf("  Path: %s\n", path)
			if cfg.DefaultOutputDevice != "" {
				fmt.Printf("  Default Output: %s\n", cfg.DefaultOutputDevice)
			}
			if cfg.DefaultInputDevice != "" {
				fmt.Printf("  Default Input: %s\n", cfg.DefaultInputDevice)
			}
			if cfg.ActiveProfile != "" {
				fmt.Printf("  Active Profile: %s\n", cfg.ActiveProfile)
			}
			fmt.Printf("  Profiles: %d\n", len(cfg.Profiles))
			
			// Show keybindings
			defaults := config.DefaultKeybindings()
			kb := cfg.Keybindings.Merge(defaults)
			fmt.Println("\nKeybindings:")
			fmt.Printf("  Quit: %s\n", kb.Quit)
			fmt.Printf("  Navigate Up: %s\n", kb.NavUp)
			fmt.Printf("  Navigate Down: %s\n", kb.NavDown)
			fmt.Printf("  Navigate Top: %s\n", kb.NavTop)
			fmt.Printf("  Navigate Bottom: %s\n", kb.NavBottom)
			fmt.Printf("  Set Default: %s\n", kb.SetDefault)
			fmt.Printf("  Volume Up: %s\n", kb.VolumeUp)
			fmt.Printf("  Volume Down: %s\n", kb.VolumeDown)
			fmt.Printf("  Volume Fine Up: %s\n", kb.VolumeFineUp)
			fmt.Printf("  Volume Fine Down: %s\n", kb.VolumeFineDown)
			fmt.Printf("  Toggle Mute: %s\n", kb.ToggleMute)
			fmt.Printf("  Tab Output: %s\n", kb.TabOutput)
			fmt.Printf("  Tab Input: %s\n", kb.TabInput)
			
			return nil
		},
	}
}

func configKeybindingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keybindings",
		Short: "Manage keybindings",
		Long:  "Set custom keybindings for the TUI.",
	}
	cmd.AddCommand(
		configKeybindingsSetCmd(),
		configKeybindingsResetCmd(),
	)
	return cmd
}

func configKeybindingsSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <action> <key>",
		Short: "Set a keybinding",
		Long: "Set a keybinding for an action.\n" +
			"Actions: quit, nav_up, nav_down, nav_top, nav_bottom, set_default, " +
			"volume_up, volume_down, volume_fine_up, volume_fine_down, toggle_mute, tab_output, tab_input",
		Example: "  audeck config keybindings set quit q\n" +
			"  audeck config keybindings set volume_up l",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			action := args[0]
			key := args[1]

			switch action {
			case "quit":
				cfg.Keybindings.Quit = key
			case "nav_up":
				cfg.Keybindings.NavUp = key
			case "nav_down":
				cfg.Keybindings.NavDown = key
			case "nav_top":
				cfg.Keybindings.NavTop = key
			case "nav_bottom":
				cfg.Keybindings.NavBottom = key
			case "set_default":
				cfg.Keybindings.SetDefault = key
			case "volume_up":
				cfg.Keybindings.VolumeUp = key
			case "volume_down":
				cfg.Keybindings.VolumeDown = key
			case "volume_fine_up":
				cfg.Keybindings.VolumeFineUp = key
			case "volume_fine_down":
				cfg.Keybindings.VolumeFineDown = key
			case "toggle_mute":
				cfg.Keybindings.ToggleMute = key
			case "tab_output":
				cfg.Keybindings.TabOutput = key
			case "tab_input":
				cfg.Keybindings.TabInput = key
			default:
				return fmt.Errorf("unknown action: %s", action)
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Keybinding '%s' set to '%s'\n", action, key)
			return nil
		},
	}
	return cmd
}

func configKeybindingsResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset keybindings to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			cfg.Keybindings = config.Keybindings{}
			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Println("Keybindings reset to defaults")
			return nil
		},
	}
}

func configThemeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Manage color themes",
		Long:  "View and change the TUI color theme.",
	}
	cmd.AddCommand(
		configThemeListCmd(),
		configThemeSetCmd(),
		configThemePreviewCmd(),
	)
	return cmd
}

func configThemeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available themes",
		RunE: func(cmd *cobra.Command, args []string) error {
			themes := []config.Theme{
				config.ThemeCatppuccinMocha,
				config.ThemeCatppuccinMacchiato,
				config.ThemeCatppuccinFrappe,
				config.ThemeCatppuccinLatte,
				config.ThemeDracula,
				config.ThemeNord,
				config.ThemeGruvbox,
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			fmt.Println("Available themes:")
			for _, theme := range themes {
				marker := " "
				if cfg.Theme == theme {
					marker = "*"
				}
				fmt.Printf("  %s %s\n", marker, string(theme))
			}
			fmt.Println("\n* = current theme")
			return nil
		},
	}
}

func configThemeSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <name>",
		Short: "Set the active theme",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			theme := config.Theme(args[0])
			// Validate theme
			_ = theme.GetColors()

			cfg.Theme = theme
			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Theme set to '%s'\n", args[0])
			fmt.Println("Restart audeck TUI to see changes")
			return nil
		},
	}
}

func configThemePreviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preview <name>",
		Short: "Preview a theme",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			theme := config.Theme(args[0])
			colors := theme.GetColors()

			fmt.Printf("Theme: %s\n\n", args[0])
			fmt.Printf("  Base:      %s ████\n", colors.Base)
			fmt.Printf("  Mantle:    %s ████\n", colors.Mantle)
			fmt.Printf("  Crust:     %s ████\n", colors.Crust)
			fmt.Printf("  Surface0:  %s ████\n", colors.Surface0)
			fmt.Printf("  Surface1:  %s ████\n", colors.Surface1)
			fmt.Printf("  Surface2:  %s ████\n", colors.Surface2)
			fmt.Printf("  Overlay0:  %s ████\n", colors.Overlay0)
			fmt.Printf("  Overlay1:  %s ████\n", colors.Overlay1)
			fmt.Printf("  Text:      %s ████\n", colors.Text)
			fmt.Printf("  Lavender:  %s ████\n", colors.Lavender)
			fmt.Printf("  Blue:      %s ████\n", colors.Blue)
			fmt.Printf("  Green:     %s ████\n", colors.Green)
			fmt.Printf("  Yellow:    %s ████\n", colors.Yellow)
			fmt.Printf("  Peach:     %s ████\n", colors.Peach)
			fmt.Printf("  Maroon:    %s ████\n", colors.Maroon)
			fmt.Printf("  Red:       %s ████\n", colors.Red)
			fmt.Printf("  Mauve:     %s ████\n", colors.Mauve)
			fmt.Printf("  Pink:      %s ████\n", colors.Pink)

			return nil
		},
	}
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputDeviceList(devices []audio.Device, filter string) error {
	if len(devices) == 0 {
		fmt.Println("No devices found")
		return nil
	}

	for _, d := range devices {
		defaultMarker := " "
		if d.IsDefault {
			defaultMarker = "*"
		}

		fmt.Printf("%s [%d] %s", defaultMarker, d.ID, d.Name)

		if d.IsOutput && d.IsInput {
			fmt.Printf(" (Input/Output)")
		} else if d.IsInput {
			fmt.Printf(" (Input)")
		}

		fmt.Println()

		if d.HasVolume {
			fmt.Printf("    Volume: %d%%", d.VolumePercent())
			if d.Muted {
				fmt.Printf(" (Muted)")
			}
			fmt.Println()
		}

		if d.TransportType != audio.TransportUnknown {
			fmt.Printf("    Transport: %s\n", d.TransportType)
		}
	}

	if filter == "all" {
		fmt.Println("\n* = default device")
	}

	return nil
}
