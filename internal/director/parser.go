package director

import (
	"encoding/json"
	"os"
)

type KeyFrame struct {
	Hour          int     `json:"hour"`
	AccelModifier float64 `json:"accel_modifier"`
	TempModifier  float64 `json:"temp_modifier"`
	DemoLat       float64 `json:"demo_lat,omitempty"`
	DemoLon       float64 `json:"demo_lon,omitempty"`
	SimSwap       bool    `json:"sim_swap,omitempty"`
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

// GetDemoOverrides returns the interpolated lat/lon and sim_swap for the current hour.
// If the scenario contains lat/lon keyframes, it interpolates them linearly.
// sim_swap becomes true from the first keyframe where it's set.
func GetDemoOverrides(currentHour float64, scenario Scenario) (float64, float64, bool) {
	if len(scenario.KeyFrames) == 0 {
		return 0, 0, false
	}

	// ---- lat/lon interpolation ----
	var lat, lon float64
	startLat, startLon := 0.0, 0.0
	endLat, endLon := 0.0, 0.0
	startHour := 0.0
	endHour := 0.0
	foundInterval := false

	// find first keyframe that has lat/lon
	firstLatIdx := -1
	for i := range scenario.KeyFrames {
		if scenario.KeyFrames[i].DemoLat != 0 || scenario.KeyFrames[i].DemoLon != 0 {
			firstLatIdx = i
			break
		}
	}
	if firstLatIdx == -1 {
		goto simSwap
	}

	// before first lat keyframe -> use the first lat keyframe values
	if currentHour <= float64(scenario.KeyFrames[firstLatIdx].Hour) {
		lat = scenario.KeyFrames[firstLatIdx].DemoLat
		lon = scenario.KeyFrames[firstLatIdx].DemoLon
		goto simSwap
	}

	for i := firstLatIdx; i < len(scenario.KeyFrames)-1; i++ {
		if scenario.KeyFrames[i+1].DemoLat != 0 || scenario.KeyFrames[i+1].DemoLon != 0 {
			startLat = scenario.KeyFrames[i].DemoLat
			startLon = scenario.KeyFrames[i].DemoLon
			endLat = scenario.KeyFrames[i+1].DemoLat
			endLon = scenario.KeyFrames[i+1].DemoLon
			startHour = float64(scenario.KeyFrames[i].Hour)
			endHour = float64(scenario.KeyFrames[i+1].Hour)
			if currentHour >= startHour && currentHour <= endHour {
				foundInterval = true
				break
			}
		}
	}

	if foundInterval {
		progress := (currentHour - startHour) / (endHour - startHour)
		lat = startLat + progress*(endLat-startLat)
		lon = startLon + progress*(endLon-startLon)
	} else {
		// past last lat keyframe -> use last known lat/lon
		for i := len(scenario.KeyFrames) - 1; i >= 0; i-- {
			if scenario.KeyFrames[i].DemoLat != 0 {
				lat = scenario.KeyFrames[i].DemoLat
				lon = scenario.KeyFrames[i].DemoLon
				break
			}
		}
	}

simSwap:
	// ---- sim_swap: becomes true from the first keyframe where it's true ----
	simSwap := false
	for i := 0; i < len(scenario.KeyFrames); i++ {
		if currentHour >= float64(scenario.KeyFrames[i].Hour) {
			if scenario.KeyFrames[i].SimSwap {
				simSwap = true
			}
		}
	}

	return lat, lon, simSwap
}