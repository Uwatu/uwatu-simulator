package emitter

import "time"

// Location holds the spatial data provided by the network
type Location struct {
	Lat               float64 `json:"lat"`
	Lon               float64 `json:"lon"`
	UncertaintyRadius int     `json:"uncertainty_radius"`
}

// NokiaSignals represents the network-derived telemetry
type NokiaSignals struct {
	DeviceLocation              Location `json:"device_location"`
	DeviceStatus                string   `json:"device_status"`
	LastSeenTimestamp           string   `json:"last_seen_timestamp"`
	SimSwapDetected             bool     `json:"sim_swap_detected"`
	SimSwapTimestamp            *string  `json:"sim_swap_timestamp"`
	ConnectivityLost            bool     `json:"connectivity_lost"`
	ConnectivityDurationSeconds int      `json:"connectivity_duration_seconds"`
	RsrpDbm                     int      `json:"rsrp_dbm"`
	ThroughputKbps              int      `json:"throughput_kbps"`
	LatencyMs                   int      `json:"latency_ms"`
	ConnectionType              string   `json:"connection_type"`
	RoamingActive               bool     `json:"roaming_active"`
	RoamingNetworkPlmn          *string  `json:"roaming_network_plmn"`
	CellCongestionLevel         int      `json:"cell_congestion_level"`
	AffectedCellIds             []string `json:"affected_cell_ids"`
	QodSessionActive            bool     `json:"qod_session_active"`
	NumberVerified              bool     `json:"number_verified"`
}

// FirmwarePayload represents the physical sensor data from the ear tag
type FirmwarePayload struct {
	AccelMagnitude int     `json:"accel_magnitude"`
	BodyTempC      float64 `json:"body_temp_c"`
	BatteryMv      int     `json:"battery_mv"`
	BatteryPct     int     `json:"battery_pct"`
	UptimeS        int     `json:"uptime_s"`
	SimTrayEvent   bool    `json:"sim_tray_event"`
	Seq            int     `json:"seq"`
}

// Context represents categorical variables for the ML model
type Context struct {
	IsNight                       bool `json:"is_night"`
	IsDrySeason                   bool `json:"is_dry_season"`
	MarketDay                     bool `json:"market_day"`
	MinutesSinceGeofenceDeparture *int `json:"minutes_since_geofence_departure"`
}

// SignalMatrix is the master payload sent to the intelligence layer
type SignalMatrix struct {
	DeviceID        string          `json:"device_id"`
	Msisdn          string          `json:"msisdn"`
	FarmID          string          `json:"farm_id"`
	AnimalID        string          `json:"animal_id"`
	Timestamp       string          `json:"timestamp"`
	NokiaSignals    NokiaSignals    `json:"nokia_signals"`
	FirmwarePayload FirmwarePayload `json:"firmware_payload"`
	Context         Context         `json:"context"`
}

// BuildNormalSignalMatrix creates a baseline JSON payload for a healthy animal.
// We pass the raw values to avoid a circular dependency between the hardware and emitter packages.
func BuildNormalSignalMatrix(deviceID, msisdn, farmID, animalID string, batteryMv, batteryPct, uptimeS, seq, accel int, temp float64, simTime time.Time) SignalMatrix {

	// Assign it to a variable called 'timeStr'
	timeStr := simTime.Format(time.RFC3339)

	// Map the incoming parameters to the correct fields.
	// For the NokiaSignals, just hardcode a "Healthy" baseline (e.g., DeviceStatus: "REACHABLE", RsrpDbm: -95).
	// For the Context, hardcode a baseline (e.g., IsNight: false).
	// In Go, the hour is 0-23. Night is roughly before 6 AM or after 6 PM.
	isNight := simTime.Hour() < 6 || simTime.Hour() >= 18
	return SignalMatrix{
		DeviceID:  deviceID,
		Msisdn:    msisdn,
		FarmID:    farmID,
		AnimalID:  animalID,
		Timestamp: timeStr,

		NokiaSignals: NokiaSignals{
			DeviceLocation:              Location{Lat: -33.789, Lon: 26.421, UncertaintyRadius: 200},
			DeviceStatus:                "REACHABLE",
			LastSeenTimestamp:           timeStr,
			SimSwapDetected:             false,
			SimSwapTimestamp:            nil,
			ConnectivityLost:            false,
			ConnectivityDurationSeconds: int(uptimeS),
			RsrpDbm:                     -80,
			ThroughputKbps:              1000,
			LatencyMs:                   20,
			ConnectionType:              "normal",
			RoamingActive:               false,
			RoamingNetworkPlmn:          nil,
			CellCongestionLevel:         0,
			AffectedCellIds:             []string{},
			QodSessionActive:            false,
			NumberVerified:              false,
		},

		FirmwarePayload: FirmwarePayload{

			AccelMagnitude: accel,
			BodyTempC:      temp,
			BatteryMv:      batteryMv,
			BatteryPct:     batteryPct,
			UptimeS:        uptimeS,
			SimTrayEvent:   false,
			Seq:            seq,
		},

		Context: Context{
			IsNight:                       isNight,
			IsDrySeason:                   true,
			MarketDay:                     true,
			MinutesSinceGeofenceDeparture: nil,
		},
	}
}
