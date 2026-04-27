package director

import (
	"encoding/json"
	"os"
)

// It needs three fields: Hour (int), AccelModifier (float64), and TempModifier (float64).
// Don't forget to add the struct tags (e.g., `json:"hour"`) so the unmarshaler knows how to map them!

type KeyFrame struct {
	Hour          int     `json:"hour"`
	AccelModifier float64 `json:"accel_modifier"`
	TempModifier  float64 `json:"temp_modifier"`
}

// It needs two fields: ScenarioName (string) and Keyframes (a slice of Keyframe structs).
// Add the `json:"scenario_name"` and `json:"keyframes"` struct tags.

type Scenario struct {
	ScenarioName string     `json:"scenario_name"`
	KeyFrames    []KeyFrame `json:"keyframes"`
}

// LoadScenario reads a JSON file from the given path and converts it into a Scenario struct.
func LoadScenario(filePath string) (Scenario, error) {
	// 1. Initialize an empty Scenario struct to hold our data.
	var scenarioData Scenario = Scenario{}

	// Use os.ReadFile(filePath).
	// It returns the file bytes and an error. Capture them both (e.g., fileBytes, err := ...)

	fileByte, err := os.ReadFile(filePath)

	// If err != nil, return an empty Scenario{} and the err.
	if err != nil {
		return Scenario{}, err
	}

	// Use json.Unmarshal(). Pass your fileBytes as the first argument,
	// and a pointer to scenarioData (&scenarioData) as the second argument.
	// This also returns an error. Capture it!

	err = json.Unmarshal(fileByte, &scenarioData)

	// If err != nil, return an empty Scenario{} and the err.
	if err != nil {
		return Scenario{}, err
	}

	return scenarioData, nil
}

// GetModifiers calculates the current modifiers based on the simulated time.
func GetModifiers(currentHour float64, scenario Scenario) (float64, float64) {
	// If there are no keyframes, return default "Healthy" modifiers (1.0x accel, 0.0 temp)
	if len(scenario.KeyFrames) == 0 {
		return 1.0, 0.0
	}

	// 1. Find the two keyframes we are between
	var startFrame, endFrame KeyFrame

	// Default to the first frame
	startFrame = scenario.KeyFrames[0]
	endFrame = scenario.KeyFrames[0]

	for i := 0; i < len(scenario.KeyFrames)-1; i++ {
		if currentHour >= float64(scenario.KeyFrames[i].Hour) && currentHour <= float64(scenario.KeyFrames[i+1].Hour) {
			startFrame = scenario.KeyFrames[i]
			endFrame = scenario.KeyFrames[i+1]
			break
		}
	}

	// 2. If we are past the last keyframe, just stay at the last known state
	if currentHour >= float64(scenario.KeyFrames[len(scenario.KeyFrames)-1].Hour) {
		last := scenario.KeyFrames[len(scenario.KeyFrames)-1]
		return last.AccelModifier, last.TempModifier
	}

	// Formula: (currentHour - startHour) / (endHour - startHour)
	// Example: (Hour 12 - Hour 0) / (Hour 24 - Hour 0) = 0.5 (50% through the transition)
	// Make sure to cast hours to float64!

	progress := (currentHour - float64(startFrame.Hour)) / (float64(endFrame.Hour) - float64(startFrame.Hour))

	// Formula: startAccel + (progress * (endAccel - startAccel))
	accelMod := startFrame.AccelModifier + (progress * (endFrame.AccelModifier - startFrame.AccelModifier))

	// Formula: startTemp + (progress * (endTemp - startTemp))
	tempMod := startFrame.TempModifier + (progress * (endFrame.TempModifier - startFrame.TempModifier))

	return accelMod, tempMod
}
