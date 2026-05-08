package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"uwatu-simulator/internal/director"
	"uwatu-simulator/internal/emitter"
	"uwatu-simulator/internal/engine"
	"uwatu-simulator/internal/hardware"
	"uwatu-simulator/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	speedFlag := flag.Int("speed", 1, "Simulation speed multiplier (e.g., 3600 for 1s = 1h)")
	scenarioFlag := flag.String("scenario", "", "Path to the scenario JSON file (optional)")
	brokerFlag := flag.String("broker", "tcp://broker.hivemq.com:1883", "MQTT broker URL")
	clientFlag := flag.String("client", "uwatu_sim_alpha_001", "MQTT Client ID")
	flag.Parse()

	myHerd := []*hardware.Tag{
		hardware.NewTag("DEV_001", "+99999991000", "FARM_NORTH", "UWT‑ZA‑COW-0001"),
		hardware.NewTag("DEV_002", "+99999991001", "FARM_NORTH", "UWT‑ZA‑COW-0002"),
	}

	startTime := time.Now()

	var scn director.Scenario
	var err error
	if *scenarioFlag != "" {
		scn, err = director.LoadScenario(*scenarioFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load scenario: %v\n", err)
			os.Exit(1)
		}
	}

	herdEmitter, err := emitter.NewMqttEmitter(*brokerFlag, *clientFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to MQTT broker: %v\n", err)
		os.Exit(1)
	}

	snapshots := make(chan engine.TagSnapshot, 64)

	simEngine := engine.NewEngine(myHerd, startTime, *speedFlag, scn, herdEmitter, snapshots)

	// Start simulation loop in background
	go func() {
		tick := 100 * time.Millisecond
		for {
			simEngine.Step(0.1)
			time.Sleep(tick)
		}
	}()

	// Pass simEngine into the Dashboard so the UI can control it
	dashboard := ui.NewDashboard(snapshots, *scenarioFlag, *speedFlag, *brokerFlag, simEngine)
	p := tea.NewProgram(dashboard, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}