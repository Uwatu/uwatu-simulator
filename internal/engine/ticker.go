package engine

import (
	"fmt"
	"time"
	"uwatu-simulator/internal/biology"
	"uwatu-simulator/internal/director"
	"uwatu-simulator/internal/emitter"
	"uwatu-simulator/internal/hardware"
)

type Engine struct {
	Tags         []*hardware.Tag
	SimTime      time.Time
	StartSimTime time.Time
	SpeedMult    int
	Scenario     director.Scenario
	Emitter      *emitter.MqttEmitter
}

func NewEngine(tags []*hardware.Tag, startTime time.Time, speedMult int, scn director.Scenario, mqttClient *emitter.MqttEmitter) *Engine {
	return &Engine{
		Tags:         tags,
		SimTime:      startTime,
		StartSimTime: startTime,
		SpeedMult:    speedMult,
		Scenario:     scn,
		Emitter:      mqttClient,
	}
}

func (e *Engine) Step(realDeltaSeconds int) {
	simulatedDeltaSeconds := realDeltaSeconds * e.SpeedMult
	e.SimTime = e.SimTime.Add(time.Duration(simulatedDeltaSeconds) * time.Second)

	// Elapsed time in hours since simulation start
	duration := e.SimTime.Sub(e.StartSimTime)
	elapsedHours := duration.Hours()

	// Disease / biology modifiers from scenario
	accelMod, tempMod := director.GetModifiers(elapsedHours, e.Scenario)

	// Location and SIM swap overrides from scenario
	demoLat, demoLon, simSwap := director.GetDemoOverrides(elapsedHours, e.Scenario)

	for _, tag := range e.Tags {
		tag.Tick(simulatedDeltaSeconds)

		// Generate baseline (healthy) sensor data
		accel, temp := biology.GenerateBaselineTelemetry(e.SimTime)

		// Apply disease modifiers
		moddedAccel := accelMod * float64(accel)
		moddedTemp := tempMod + temp

		// Build the base signal matrix
		payload := emitter.BuildNormalSignalMatrix(
			tag.DeviceID, tag.Msisdn, tag.FarmID, tag.AnimalID,
			tag.BatteryMv, tag.BatteryPct, tag.UptimeS, tag.Seq,
			int(moddedAccel), moddedTemp, e.SimTime,
		)

		// Override with scenario location if provided
		if demoLat != 0 && demoLon != 0 {
			payload.DemoLat = demoLat
			payload.DemoLon = demoLon
		}
		payload.SimSwap = simSwap

		// Publish to MQTT
		err := e.Emitter.Publish(tag.FarmID, tag.DeviceID, payload)
		if err != nil {
			fmt.Printf("Failed to publish for tag %s: %v\n", tag.DeviceID, err)
		} else {
			locStr := "no scenario"
if demoLat != 0 && demoLon != 0 {
    locStr = fmt.Sprintf("%.4f, %.4f", demoLat, demoLon)
}
swapStr := "false"
if simSwap {
    swapStr = "true"
}

fmt.Printf("[MQTT] %-8s │ TEMP: %4.1f°C │ ACC: %3d │ BATT: %3d%% │ LOC: %s │ SIM_SWAP: %s\n",
    tag.DeviceID,
    moddedTemp,
    int(moddedAccel),
    tag.BatteryPct,
    locStr,
    swapStr,
)
		}
	}
}