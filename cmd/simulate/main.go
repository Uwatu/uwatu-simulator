package main

import (
	"flag"
	"fmt"
	"time"
	"uwatu-simulator/internal/director"
	"uwatu-simulator/internal/emitter"
	"uwatu-simulator/internal/engine"
	"uwatu-simulator/internal/hardware"
)

func main() {
	speedFlag := flag.Int("speed", 1, "Simulation speed multiplier (e.g., 3600 for 1s = 1h)")
	scenarioFlag := flag.String("scenario", "", "Path to the scenario JSON file (optional)")
	brokerFlag := flag.String("broker", "tcp://broker.hivemq.com:1883", "MQTT broker URL")
	clientFlag := flag.String("client", "uwatu_sim_alpha_001", "MQTT Client ID")

	// 2. Parse the flags from the terminal input
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("  Uwatu Digital Twin Simulator v1.0.0")
	fmt.Println("========================================")

	myHerd := []*hardware.Tag{
		hardware.NewTag("DEV_001", "+99999991000", "FARM_NORTH", "UWT‑ZA‑COW-0001"),
		hardware.NewTag("DEV_002", "+99999991001", "FARM_NORTH", "UWT‑ZA‑COW-0002"),
	}

	startTime := time.Now()

	// 3. Load Scenario (Only if a path was provided)
	var scn director.Scenario
	var err error
	if *scenarioFlag != "" {
		scn, err = director.LoadScenario(*scenarioFlag)
		if err != nil {
			panic(fmt.Sprintf("Failed to load scenario: %v", err))
		}
		fmt.Printf("[CONFIG] Loaded Scenario: %s\n", scn.ScenarioName)
	} else {
		fmt.Println("[CONFIG] Running Default Healthy Baseline")
	}

	// 4. Initialize Network
	fmt.Printf("[NETWORK] Connecting to Broker: %s\n", *brokerFlag)
	herdEmitter, err := emitter.NewMqttEmitter(*brokerFlag, *clientFlag)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MQTT broker: %v", err))
	}

	// 5. Initialize Engine with Flag Values (Dereference pointers with *)
	simEngine := engine.NewEngine(myHerd, startTime, *speedFlag, scn, herdEmitter)

	fmt.Printf("[SYSTEM] Engine online. Running at %dx speed.\n", *speedFlag)
	fmt.Println("Press Ctrl+C to terminate.")
	fmt.Println("----------------------------------------")

	// 6. The Core Loop
	for {
		simEngine.Step(1)
		time.Sleep(1 * time.Second)
	}
}
