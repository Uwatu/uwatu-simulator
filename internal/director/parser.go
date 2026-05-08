package director

import (
	"encoding/json"
	"os"
)

type KeyFrame struct {
	Hour          float64  `json:"hour"`
	AccelModifier float64  `json:"accel_modifier"`
	TempModifier  float64  `json:"temp_modifier"`
	DemoLat       float64  `json:"demo_lat,omitempty"`
	DemoLon       float64  `json:"demo_lon,omitempty"`
	SimSwap       bool     `json:"sim_swap,omitempty"`
	Devices       []string `json:"devices,omitempty"` // empty → all devices
}

type Scenario struct {
	ScenarioName string     `json:"scenario_name"`
	KeyFrames    []KeyFrame `json:"keyframes"`
}

func LoadScenario(filePath string) (Scenario, error) {
	var scenarioData Scenario

	fileByte, err := os.ReadFile(filePath)
	if err != nil {
		return Scenario{}, err
	}

	err = json.Unmarshal(fileByte, &scenarioData)
	if err != nil {
		return Scenario{}, err
	}

	return scenarioData, nil
}

// GetModifiers calculates the current modifiers based on the simulated time.
func GetModifiers(currentHour float64, scenario Scenario) (float64, float64) {
	if len(scenario.KeyFrames) == 0 {
		return 1.0, 0.0
	}

	var startFrame, endFrame KeyFrame

	startFrame = scenario.KeyFrames[0]
	endFrame = scenario.KeyFrames[0]

	for i := 0; i < len(scenario.KeyFrames)-1; i++ {
		if currentHour >= scenario.KeyFrames[i].Hour && currentHour <= scenario.KeyFrames[i+1].Hour {
			startFrame = scenario.KeyFrames[i]
			endFrame = scenario.KeyFrames[i+1]
			break
		}
	}

	if currentHour >= scenario.KeyFrames[len(scenario.KeyFrames)-1].Hour {
		last := scenario.KeyFrames[len(scenario.KeyFrames)-1]
		return last.AccelModifier, last.TempModifier
	}

	progress := (currentHour - startFrame.Hour) / (endFrame.Hour - startFrame.Hour)

	accelMod := startFrame.AccelModifier + (progress * (endFrame.AccelModifier - startFrame.AccelModifier))
	tempMod := startFrame.TempModifier + (progress * (endFrame.TempModifier - startFrame.TempModifier))

	return accelMod, tempMod
}

// deviceMatches returns true if the keyframe applies to the given device.
func deviceMatches(frame KeyFrame, deviceID string) bool {
	if len(frame.Devices) == 0 {
		return true
	}
	for _, d := range frame.Devices {
		if d == deviceID {
			return true
		}
	}
	return false
}

// GetDemoInterpolated returns smoothly interpolated (lat, lon, sim_swap)
// for a specific device at the given hour. Position is blended between
// the two nearest keyframes; sim_swap is only true when reaching that frame.
func GetDemoInterpolated(currentHour float64, scenario Scenario, deviceID string) (float64, float64, bool) {
	if len(scenario.KeyFrames) == 0 {
		return 0, 0, false
	}

	// Collect keyframes that apply to this device (or all devices)
	var frames []KeyFrame
	for _, kf := range scenario.KeyFrames {
		if deviceMatches(kf, deviceID) {
			frames = append(frames, kf)
		}
	}
	if len(frames) == 0 {
		return 0, 0, false
	}

	// Find the segment that contains currentHour
	var prev, next KeyFrame
	prev = frames[0]
	next = frames[0]

	if currentHour <= frames[0].Hour {
		prev = frames[0]
		next = frames[0]
	} else if currentHour >= frames[len(frames)-1].Hour {
		prev = frames[len(frames)-1]
		next = frames[len(frames)-1]
	} else {
		for i := 0; i < len(frames)-1; i++ {
			if currentHour >= frames[i].Hour && currentHour <= frames[i+1].Hour {
				prev = frames[i]
				next = frames[i+1]
				break
			}
		}
	}

	// If neither frame has position data, return zero
	if (prev.DemoLat == 0 && prev.DemoLon == 0) && (next.DemoLat == 0 && next.DemoLon == 0) {
		return 0, 0, false
	}

	// If one frame lacks position, use the other directly
	if prev.DemoLat == 0 && prev.DemoLon == 0 {
		return next.DemoLat, next.DemoLon, (currentHour >= next.Hour && next.SimSwap)
	}
	if next.DemoLat == 0 && next.DemoLon == 0 {
		return prev.DemoLat, prev.DemoLon, (currentHour >= prev.Hour && prev.SimSwap)
	}

	// Compute blend factor
	span := next.Hour - prev.Hour
	var t float64
	if span > 0 {
		t = (currentHour - prev.Hour) / span
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
	} else {
		t = 0
	}

	// Linearly interpolate position
	lat := prev.DemoLat + (next.DemoLat-prev.DemoLat)*t
	lon := prev.DemoLon + (next.DemoLon-prev.DemoLon)*t

	// SIM swap becomes true only when we are exactly at or past the keyframe that sets it
	simSwap := false
	if currentHour >= next.Hour && next.SimSwap {
		simSwap = true
	}

	return lat, lon, simSwap
}

// GetDemoOverrides returns the effective (lat, lon, sim_swap) for a specific device at the given hour.
// Kept for backwards compatibility; new code should use GetDemoInterpolated for smooth motion.
func GetDemoOverrides(currentHour float64, scenario Scenario, deviceID string) (float64, float64, bool) {
	if len(scenario.KeyFrames) == 0 {
		return 0, 0, false
	}

	var lat, lon float64
	var simSwap bool
	latFound := false

	for i := len(scenario.KeyFrames) - 1; i >= 0; i-- {
		frame := scenario.KeyFrames[i]
		if currentHour >= frame.Hour && deviceMatches(frame, deviceID) {
			if !latFound && (frame.DemoLat != 0 || frame.DemoLon != 0) {
				lat = frame.DemoLat
				lon = frame.DemoLon
				latFound = true
			}
			if frame.SimSwap {
				simSwap = true
				break
			}
		}
	}

	if !latFound {
		for i := 0; i < len(scenario.KeyFrames); i++ {
			if deviceMatches(scenario.KeyFrames[i], deviceID) && (scenario.KeyFrames[i].DemoLat != 0 || scenario.KeyFrames[i].DemoLon != 0) {
				lat = scenario.KeyFrames[i].DemoLat
				lon = scenario.KeyFrames[i].DemoLon
				break
			}
		}
	}

	return lat, lon, simSwap
}