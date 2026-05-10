package engine

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"uwatu-simulator/internal/biology"
	"uwatu-simulator/internal/director"
	"uwatu-simulator/internal/emitter"
	"uwatu-simulator/internal/hardware"
)

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
	Lat        float64
	Lon        float64
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

	mu sync.RWMutex
}

// default locations for each device (Kericho farm)
var defaultLocations = map[string][2]float64{
	"DEV_001": {-0.355361, 35.305120},
	"DEV_002": {-0.360627, 35.300798},
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

func (e *Engine) SetScenario(scn director.Scenario) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Scenario = scn
	e.StartSimTime = e.SimTime
}

func (e *Engine) RestartScenario() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.StartSimTime = e.SimTime
}

func (e *Engine) SetSpeed(speed int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.SpeedMult = speed
}

func (e *Engine) Step(realDeltaSeconds float64) {
	e.mu.RLock()
	currentScenario := e.Scenario
	startSimTime := e.StartSimTime
	currentSpeed := e.SpeedMult
	e.mu.RUnlock()

	simulatedDeltaSeconds := realDeltaSeconds * float64(currentSpeed)
	e.SimTime = e.SimTime.Add(time.Duration(simulatedDeltaSeconds * float64(time.Second)))

	duration := e.SimTime.Sub(startSimTime)
	elapsedHours := duration.Hours()

	accelMod, tempMod := director.GetModifiers(elapsedHours, currentScenario)

	for _, tag := range e.Tags {
		tag.Tick(int(simulatedDeltaSeconds))

		accel, temp := biology.GenerateBaselineTelemetry(e.SimTime)

		moddedAccel := accelMod * float64(accel)
		moddedTemp := tempMod + temp

		var newAccel int = int(moddedAccel)
		if newAccel <= 0 {
			newAccel = rand.IntN(4) + 1
		}

		payload := emitter.BuildNormalSignalMatrix(
			tag.DeviceID, tag.Msisdn, tag.FarmID, tag.AnimalID,
			tag.BatteryMv, tag.BatteryPct, tag.UptimeS, tag.Seq,
			newAccel, moddedTemp, e.SimTime,
		)

		// Always get position & sim_swap from scenario (interpolated) or default
		demoLat, demoLon, simSwap := director.GetDemoInterpolated(elapsedHours, currentScenario, tag.DeviceID)

		// Fallback to device's default location if no scenario provides coordinates
		if demoLat == 0 && demoLon == 0 {
			if defaultLoc, ok := defaultLocations[tag.DeviceID]; ok {
				demoLat = defaultLoc[0]
				demoLon = defaultLoc[1]
			}
		}

		// Always write them into the payload (fields are non-omitempty)
		payload.DemoLat = demoLat
		payload.DemoLon = demoLon
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
			Lat:        demoLat,
			Lon:        demoLon,
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