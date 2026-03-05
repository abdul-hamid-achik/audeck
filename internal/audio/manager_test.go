package audio

import (
	"testing"
)

func TestClampVolume(t *testing.T) {
	tests := []struct {
		name     string
		input    float32
		expected float32
	}{
		{"normal", 0.5, 0.5},
		{"too high", 1.5, 1.0},
		{"too low", -0.5, 0.0},
		{"zero", 0.0, 0.0},
		{"max", 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampVolume(tt.input)
			if result != tt.expected {
				t.Errorf("clampVolume(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDeviceVolumePercent(t *testing.T) {
	tests := []struct {
		name     string
		volume   float32
		expected int
	}{
		{"half", 0.5, 50},
		{"full", 1.0, 100},
		{"zero", 0.0, 0},
		{"quarter", 0.25, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Device{Volume: tt.volume}
			result := d.VolumePercent()
			if result != tt.expected {
				t.Errorf("Device.VolumePercent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDeviceDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		device   Device
		expected string
	}{
		{
			"unknown transport",
			Device{Name: "Test", TransportType: TransportUnknown},
			"Test",
		},
		{
			"built-in transport",
			Device{Name: "Speakers", TransportType: TransportBuiltIn},
			"Speakers",
		},
		{
			"USB transport",
			Device{Name: "USB Mic", TransportType: TransportUSB},
			"USB Mic (USB)",
		},
		{
			"Bluetooth transport",
			Device{Name: "BT Headphones", TransportType: TransportBluetooth},
			"BT Headphones (Bluetooth)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.DisplayName()
			if result != tt.expected {
				t.Errorf("Device.DisplayName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransportTypeString(t *testing.T) {
	tests := []struct {
		transport TransportType
		expected  string
	}{
		{TransportUnknown, "Unknown"},
		{TransportBuiltIn, "Built-in"},
		{TransportUSB, "USB"},
		{TransportBluetooth, "Bluetooth"},
		{TransportHDMI, "HDMI"},
		{TransportDisplayPort, "DisplayPort"},
		{TransportAirPlay, "AirPlay"},
		{TransportThunderbolt, "Thunderbolt"},
		{TransportVirtual, "Virtual"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.transport.String()
			if result != tt.expected {
				t.Errorf("TransportType(%d).String() = %v, want %v", tt.transport, result, tt.expected)
			}
		})
	}
}

func TestDeviceScope(t *testing.T) {
	if ScopeOutput != 0 {
		t.Errorf("ScopeOutput should be 0, got %d", ScopeOutput)
	}
	if ScopeInput != 1 {
		t.Errorf("ScopeInput should be 1, got %d", ScopeInput)
	}
}
