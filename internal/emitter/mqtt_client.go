package emitter

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MqttEmitter handles the network connection to the MQTT broker
type MqttEmitter struct {
	Client mqtt.Client
}

// NewMqttEmitter initializes and connects to the MQTT broker
func NewMqttEmitter(brokerURL, clientID string) (*MqttEmitter, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)

	// Set a connection timeout to prevent the app from hanging
	opts.SetConnectTimeout(2 * time.Second)

	client := mqtt.NewClient(opts)

	clientToken := client.Connect()

	clientToken.Wait()
	if err := clientToken.Error(); err != nil {
		return nil, err
	}

	return &MqttEmitter{Client: client}, nil
}

// Publish takes the telemetry data and sends it to a specific MQTT topic
func (m *MqttEmitter) Publish(farmID, deviceID string, payload SignalMatrix) error {
	// Construct the topic string using the hierarchical pattern
	topic := fmt.Sprintf("uwatu/farm/%s/tag/%s", farmID, deviceID)

	// Handles the potential error.

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	publishToken := m.Client.Publish(topic, 1, false, payloadBytes)
	publishToken.Wait()
	if err := publishToken.Error(); err != nil {
		return err
	}

	return nil
}
