//go:build darwin

package coreaudio

import (
	"os"
	"testing"
)

func requireCoreAudioIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping CoreAudio integration test in -short mode")
	}
	if os.Getenv("AUDECK_RUN_COREAUDIO_INTEGRATION") == "" {
		t.Skip("set AUDECK_RUN_COREAUDIO_INTEGRATION=1 to run CoreAudio integration tests")
	}
}

func TestGetDefaultOutputDevice(t *testing.T) {
	requireCoreAudioIntegration(t)

	id, err := GetDefaultOutputDevice()
	if err != nil {
		t.Fatalf("GetDefaultOutputDevice: %v", err)
	}
	if id == 0 {
		t.Fatal("GetDefaultOutputDevice returned 0")
	}
	t.Logf("default output device ID: %d", id)
}

func TestGetDefaultInputDevice(t *testing.T) {
	requireCoreAudioIntegration(t)

	id, err := GetDefaultInputDevice()
	if err != nil {
		t.Fatalf("GetDefaultInputDevice: %v", err)
	}
	if id == 0 {
		t.Fatal("GetDefaultInputDevice returned 0")
	}
	t.Logf("default input device ID: %d", id)
}

func TestSetDefaultOutputDevice_InvalidID(t *testing.T) {
	err := SetDefaultOutputDevice(0)
	if err == nil {
		t.Fatal("expected error for device ID 0")
	}
	if err != ErrNoDevice {
		t.Fatalf("expected ErrNoDevice, got: %v", err)
	}
}

func TestSetDefaultInputDevice_InvalidID(t *testing.T) {
	err := SetDefaultInputDevice(0)
	if err == nil {
		t.Fatal("expected error for device ID 0")
	}
	if err != ErrNoDevice {
		t.Fatalf("expected ErrNoDevice, got: %v", err)
	}
}

func TestSetDefaultOutputDevice_RoundTrip(t *testing.T) {
	requireCoreAudioIntegration(t)

	// Get current default, set it again, verify it's unchanged.
	original, err := GetDefaultOutputDevice()
	if err != nil {
		t.Fatalf("GetDefaultOutputDevice: %v", err)
	}

	if err := SetDefaultOutputDevice(original); err != nil {
		t.Fatalf("SetDefaultOutputDevice(%d): %v", original, err)
	}

	after, err := GetDefaultOutputDevice()
	if err != nil {
		t.Fatalf("GetDefaultOutputDevice after set: %v", err)
	}
	if after != original {
		t.Fatalf("default output changed: got %d, want %d", after, original)
	}
}

func TestSetDefaultInputDevice_RoundTrip(t *testing.T) {
	requireCoreAudioIntegration(t)

	original, err := GetDefaultInputDevice()
	if err != nil {
		t.Fatalf("GetDefaultInputDevice: %v", err)
	}

	if err := SetDefaultInputDevice(original); err != nil {
		t.Fatalf("SetDefaultInputDevice(%d): %v", original, err)
	}

	after, err := GetDefaultInputDevice()
	if err != nil {
		t.Fatalf("GetDefaultInputDevice after set: %v", err)
	}
	if after != original {
		t.Fatalf("default input changed: got %d, want %d", after, original)
	}
}
