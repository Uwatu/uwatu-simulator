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

	// This returns a time.Duration (nanoseconds)
	duration := e.SimTime.Sub(e.StartSimTime)

	// This converts that duration into a float64 of hours (e.g., 0.5 for 30 mins)
	elapsedHours := duration.Hours()
	accelMod, tempMod := director.GetModifiers(float64(elapsedHours), e.Scenario)

	for _, tag := range e.Tags {
		tag.Tick(simulatedDeltaSeconds)

		// Generate the baseline (healthy) math
		accel, temp := biology.GenerateBaselineTelemetry(e.SimTime)

		moddedAccel := accelMod * float64(accel)
		moddedTemp := tempMod + float64(temp)

		var newAccel int = int(moddedAccel)
		if newAccel < 0 {
			newAccel = 0
		}

		payload := emitter.BuildNormalSignalMatrix(tag.DeviceID, tag.Msisdn, tag.FarmID, tag.AnimalID,
			tag.BatteryMv, tag.BatteryPct, tag.UptimeS, tag.Seq, newAccel, moddedTemp, e.SimTime)

		err := e.Emitter.Publish(tag.FarmID, tag.DeviceID, payload)
		if err != nil {
			fmt.Printf("Failed to publish for tag %s: %v\n", tag.DeviceID, err)
		} else {
			fmt.Printf("[MQTT] Sent payload for %s at %s\n", tag.DeviceID, e.SimTime.Format("15:04:05"))
		}

	}
}
