package engine

import (
	"fmt"
	"sync"
	"time"

	"uwatu-simulator/internal/biology"
	"uwatu-simulator/internal/director"
	"uwatu-simulator/internal/emitter"
	"uwatu-simulator/internal/hardware"
)

// TagSnapshot is a point-in-time reading for one tag, sent to the UI layer.
type TagSnapshot struct {
	DeviceID   string
	AnimalID   string
	FarmID     string
	Temp       float64
	Accel      int
	BatteryPct int
	BatteryMv  int
	UptimeS    int
	Seq        int
	DemoLat    float64
	DemoLon    float64
	SimSwap    bool
	SimTime    time.Time
	PublishErr error
}

type Engine struct {
	Tags         []*hardware.Tag
	SimTime      time.Time
	StartSimTime time.Time
	SpeedMult    int
	Scenario     director.Scenario
	Emitter      *emitter.MqttEmitter
	Snapshots    chan<- TagSnapshot

	mu sync.RWMutex // Protects Scenario and StartSimTime
}

func NewEngine(
	tags []*hardware.Tag,
	startTime time.Time,
	speedMult int,
	scn director.Scenario,
	mqttClient *emitter.MqttEmitter,
	snapshots chan<- TagSnapshot,
) *Engine {
	return &Engine{
		Tags:         tags,
		SimTime:      startTime,
		StartSimTime: startTime,
		SpeedMult:    speedMult,
		Scenario:     scn,
		Emitter:      mqttClient,
		Snapshots:    snapshots,
	}
}

// SetScenario safely swaps the active scenario and resets the scenario timeline.
func (e *Engine) SetScenario(scn director.Scenario) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Scenario = scn
	e.StartSimTime = e.SimTime // Reset so the new scenario starts at hour 0
}

func (e *Engine) Step(realDeltaSeconds float64) {
	simulatedDeltaSeconds := realDeltaSeconds * float64(e.SpeedMult)
	e.SimTime = e.SimTime.Add(time.Duration(simulatedDeltaSeconds * float64(time.Second)))

	// Safely read scenario and start time
	e.mu.RLock()
	currentScenario := e.Scenario
	startSim := e.StartSimTime
	e.mu.RUnlock()

	duration := e.SimTime.Sub(startSim)
	elapsedHours := duration.Hours()

	accelMod, tempMod := director.GetModifiers(elapsedHours, currentScenario)

	for _, tag := range e.Tags {
		tag.Tick(int(simulatedDeltaSeconds))
		accel, temp := biology.GenerateBaselineTelemetry(e.SimTime)

		moddedAccel := accelMod * float64(accel)
		moddedTemp := tempMod + temp

		var newAccel int = int(moddedAccel)
		if newAccel < 0 {
			newAccel = 0
		}

		payload := emitter.BuildNormalSignalMatrix(
			tag.DeviceID, tag.Msisdn, tag.FarmID, tag.AnimalID,
			tag.BatteryMv, tag.BatteryPct, tag.UptimeS, tag.Seq,
			newAccel, moddedTemp, e.SimTime,
		)

		demoLat, demoLon, simSwap := director.GetDemoInterpolated(elapsedHours, currentScenario, tag.DeviceID)

		if demoLat != 0 && demoLon != 0 {
			payload.DemoLat = demoLat
			payload.DemoLon = demoLon
		}
		payload.SimSwap = simSwap

		publishErr := e.Emitter.Publish(tag.FarmID, tag.DeviceID, payload)
		if publishErr != nil {
			fmt.Printf("publish error for %s: %v\n", tag.DeviceID, publishErr)
		}

		snap := TagSnapshot{
			DeviceID:   tag.DeviceID,
			AnimalID:   tag.AnimalID,
			FarmID:     tag.FarmID,
			Temp:       moddedTemp,
			Accel:      newAccel,
			BatteryPct: tag.BatteryPct,
			BatteryMv:  tag.BatteryMv,
			UptimeS:    tag.UptimeS,
			Seq:        tag.Seq,
			DemoLat:    demoLat,
			DemoLon:    demoLon,
			SimSwap:    simSwap,
			SimTime:    e.SimTime,
			PublishErr: publishErr,
		}

		select {
		case e.Snapshots <- snap:
		default:
		}
	}
}