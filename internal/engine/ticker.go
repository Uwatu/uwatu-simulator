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

	duration := e.SimTime.Sub(e.StartSimTime)
	elapsedHours := duration.Hours()

	accelMod, tempMod := director.GetModifiers(elapsedHours, e.Scenario)

	for _, tag := range e.Tags {
		tag.Tick(simulatedDeltaSeconds)

		accel, temp := biology.GenerateBaselineTelemetry(e.SimTime)

		moddedAccel := accelMod * float64(accel)
		moddedTemp := tempMod + temp

		var newAccel int = int(moddedAccel)
		if newAccel < 0 {
			newAccel = 0
		}

		payload := emitter.BuildNormalSignalMatrix(tag.DeviceID, tag.Msisdn, tag.FarmID, tag.AnimalID,
			tag.BatteryMv, tag.BatteryPct, tag.UptimeS, tag.Seq, newAccel, moddedTemp, e.SimTime)

		// Per‑device demo overrides
		demoLat, demoLon, simSwap := director.GetDemoOverrides(elapsedHours, e.Scenario, tag.DeviceID)
		if demoLat != 0 && demoLon != 0 {
			payload.DemoLat = demoLat
			payload.DemoLon = demoLon
		}
		payload.SimSwap = simSwap

		// Build a descriptive log line
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
			newAccel,
			tag.BatteryPct,
			locStr,
			swapStr,
		)

		err := e.Emitter.Publish(tag.FarmID, tag.DeviceID, payload)
		if err != nil {
			fmt.Printf("Failed to publish for tag %s: %v\n", tag.DeviceID, err)
		}
	}
}