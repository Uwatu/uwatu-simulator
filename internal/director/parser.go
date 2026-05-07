package director

import (
	"encoding/json"
	"os"
)

type KeyFrame struct {
	Hour          int      `json:"hour"`
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
		if currentHour >= float64(scenario.KeyFrames[i].Hour) && currentHour <= float64(scenario.KeyFrames[i+1].Hour) {
			startFrame = scenario.KeyFrames[i]
			endFrame = scenario.KeyFrames[i+1]
			break
		}
	}

	if currentHour >= float64(scenario.KeyFrames[len(scenario.KeyFrames)-1].Hour) {
		last := scenario.KeyFrames[len(scenario.KeyFrames)-1]
		return last.AccelModifier, last.TempModifier
	}

	progress := (currentHour - float64(startFrame.Hour)) / (float64(endFrame.Hour) - float64(startFrame.Hour))

	accelMod := startFrame.AccelModifier + (progress * (endFrame.AccelModifier - startFrame.AccelModifier))
	tempMod := startFrame.TempModifier + (progress * (endFrame.TempModifier - startFrame.TempModifier))

	return accelMod, tempMod
}

// deviceMatches returns true if the keyframe applies to the given device.
func deviceMatches(frame KeyFrame, deviceID string) bool {
	if len(frame.Devices) == 0 {
		return true // applies to all
	}
	for _, d := range frame.Devices {
		if d == deviceID {
			return true
		}
	}
	return false
}

// GetDemoOverrides returns the effective (lat, lon, sim_swap) for a specific device at the given hour.
// It finds the last keyframe before or at currentHour whose Devices contain the deviceID
// (or which has no Devices, meaning all devices). No interpolation – uses the exact values of
// that keyframe.
func GetDemoOverrides(currentHour float64, scenario Scenario, deviceID string) (float64, float64, bool) {
	if len(scenario.KeyFrames) == 0 {
		return 0, 0, false
	}

	var lat, lon float64
	var simSwap bool
	latFound := false

	for i := len(scenario.KeyFrames) - 1; i >= 0; i-- {
		frame := scenario.KeyFrames[i]
		if currentHour >= float64(frame.Hour) && deviceMatches(frame, deviceID) {
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