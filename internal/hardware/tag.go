package hardware

import (
	"slices"
	"uwatu-simulator/internal/emitter"
)

// Tag represents the in-memory state of a single physical ear tag.
type Tag struct {
	DeviceID string
	Msisdn   string
	FarmID   string
	AnimalID string

	// Internal Hardware State
	BatteryMv  int
	BatteryPct int
	UptimeS    int
	Seq        int

	// Virtual Flash Memory for Graceful Degradation
	FlashBuffer []emitter.SignalMatrix
}

// NewTag initializes a fresh tag.
func NewTag(deviceID, msisdn, farmID, animalID string) *Tag {
	return &Tag{
		DeviceID:    deviceID,
		Msisdn:      msisdn,
		FarmID:      farmID,
		AnimalID:    animalID,
		BatteryMv:   4100, // Simulating a fully charged battery (4.1V)
		BatteryPct:  100,
		UptimeS:     0,
		Seq:         0,
		FlashBuffer: make([]emitter.SignalMatrix, 0), // Initialize empty slice
	}
}

// Tick simulates the passage of time on the hardware.
// deltaSeconds is how much simulated time has passed since the last tick.
func (t *Tag) Tick(deltaSeconds int) {
	t.UptimeS += deltaSeconds
	t.Seq++

	// 1. Calculate how many hours have passed
	hoursPassed := deltaSeconds / 3600

	if hoursPassed > 0 {
		t.BatteryMv -= hoursPassed

		// Recalculate the percentage (mapping 3000mV - 4100mV to 0-100%)
		pct := ((t.BatteryMv - 3000) * 100) / 1100

		// Clamp the percentage to ensure it never goes out of bounds
		if pct > 100 {
			t.BatteryPct = 100
		} else if pct < 0 {
			t.BatteryPct = 0
		} else {
			t.BatteryPct = pct
		}
	}
}

// BufferMessage saves a payload to the tag's memory during a network outage.
func (t *Tag) BufferMessage(payload emitter.SignalMatrix) {
	t.FlashBuffer = append(t.FlashBuffer, payload)
}

// FlushBuffer extracts all saved payloads and clears the memory.
func (t *Tag) FlushBuffer() []emitter.SignalMatrix {

	currentBuffer := slices.Clone(t.FlashBuffer)

	t.FlashBuffer = make([]emitter.SignalMatrix, 0)

	return currentBuffer
}
