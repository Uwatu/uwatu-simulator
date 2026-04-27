# Uwatu Digital Twin Simulator

The Uwatu Simulator is a high-fidelity IoT digital twin engine built in Go. It generates biologically accurate, network-transmitted telemetry data for livestock tracking ear tags.

Unlike standard hardware simulators that output random noise, Uwatu utilizes a Biological Math Engine and a Scenario Director to simulate circadian rhythms, hardware degradation, and multi-day disease outbreaks. This provides a clean, labeled "Ground Truth" dataset for training downstream Machine Learning (XGBoost) anomaly detection models.

## Core Features

* **Biological Emulation:** Simulates diurnal temperature cycles (37.5 C - 39.5 C) and bimodal grazing activity curves (dawn/dusk movement spikes).
* **Time-Warp Engine:** Accelerate time to simulate 72 hours of herd behavior in a matter of minutes.
* **Scenario Director:** Inject custom JSON scenarios using Linear Interpolation to smoothly transition a healthy herd into a disease outbreak or theft event.
* **Hardware Degradation:** Simulates device uptime, sequence tracking, and voltage drop over simulated hours.
* **Live Network Emitter:** Broadcasts data natively over MQTT with QoS 1 to any standard broker, acting exactly like physical hardware in the field.

## Architecture

1. **Biology Package:** Generates the healthy baseline using Gaussian noise and sine waves.
2. **Director Package:** Parses JSON keyframes and calculates behavioral modifiers.
3. **Hardware Package:** Manages physical device states (battery, uptime).
4. **Engine Package:** The core loop that merges biology, hardware, and scenarios.
5. **Emitter Package:** The Paho MQTT client that publishes the SignalMatrix payload.

## Installation and Build

Ensure you have Go installed, then clone the repository:

git clone [https://github.com/uwatu/uwatu-simulator.git](https://github.com/uwatu/uwatu-simulator.git)
cd uwatu-simulator

**Install Dependencies:**
The simulator relies on the Eclipse Paho MQTT library.
go get [github.com/eclipse/paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang)

**Build the Executable:**
For Linux/macOS:
go build -o bin/uwatu-simulator cmd/simulate/main.go

For Windows:
go build -o bin/uwatu-simulator.exe cmd/simulate/main.go

## Usage and CLI Flags

The simulator is fully dynamic and controlled via terminal flags.

**View all commands:**
./bin/uwatu-simulator --help

**Run a baseline (Healthy) simulation at normal speed:**
./bin/uwatu-simulator

**Run a Time-Warped Disease Scenario (1 real second = 1 simulated hour):**
./bin/uwatu-simulator --speed 3600 --scenario config/scenarios/disease_72h.json

**Point to a Custom MQTT Broker:**
./bin/uwatu-simulator --broker tcp://localhost:1883 --client my_custom_sim_01

## Creating Scenarios

Scenarios dictate how the herd's biology deviates from the baseline over time. Create a JSON file in config/scenarios/ using the Keyframe structure. The engine will automatically interpolate the values between hours.

**Example: disease_72h.json**
{
"scenario_name": "Rapid Disease Outbreak",
"keyframes": [
{ "hour": 0, "accel_modifier": 1.0, "temp_modifier": 0.0 },
{ "hour": 24, "accel_modifier": 0.7, "temp_modifier": 0.8 },
{ "hour": 72, "accel_modifier": 0.3, "temp_modifier": 2.1 }
]
}

## Data Output (Signal Matrix)

The simulator publishes telemetry data to the broker using a hierarchical topic structure: uwatu/farm/{farmID}/tag/{deviceID}.

**Sample JSON Payload:**
{
"device_id": "DEV_001",
"msisdn": "27830000001",
"farm_id": "FARM_NORTH",
"animal_id": "COW_A1",
"battery_mv": 3580,
"battery_pct": 98.5,
"uptime_s": 86400,
"seq": 142,
"accel_magnitude": 12,
"body_temp_c": 38.6,
"timestamp": "2026-04-26T18:00:00Z"
}

## Watching the Live Stream
By default, the simulator targets the HiveMQ public sandbox. You can watch your data stream live by visiting the HiveMQ Websocket Client and subscribing to the topic: uwatu/farm/+/tag/#