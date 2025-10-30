package led

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"warehousecore/internal/models"
)

// ControllerListener subscribes to controller heartbeat topics and auto-registers devices.
type ControllerListener struct {
	client      mqtt.Client
	config      PublisherConfig
	topicFilter string
	dryRun      bool
	startOnce   sync.Once
}

// controllerHeartbeatHandler is assigned from main so we avoid package import cycles.
var controllerHeartbeatHandler func(identifier string, payload *models.LEDControllerHeartbeat) error

// RegisterControllerHeartbeatHandler installs the callback used when a controller heartbeat is received.
func RegisterControllerHeartbeatHandler(handler func(identifier string, payload *models.LEDControllerHeartbeat) error) {
	controllerHeartbeatHandler = handler
}

// NewControllerListener creates the listener but does not start it yet.
func NewControllerListener() *ControllerListener {
	cfg := buildMQTTConfig("warehousecore-listener")
	topicPrefix := strings.Trim(cfg.TopicPrefix, "/")
	if topicPrefix == "" {
		topicPrefix = "weidelbach"
	}

	listener := &ControllerListener{
		config:      cfg,
		topicFilter: fmt.Sprintf("%s/+/status", topicPrefix),
	}

	if cfg.Host == "" {
		listener.dryRun = true
		log.Println("[LED] Controller listener disabled (LED_MQTT_HOST not configured)")
	}

	return listener
}

// Start begins listening for controller heartbeats.
func (l *ControllerListener) Start() {
	l.startOnce.Do(func() {
		if l.dryRun {
			return
		}
		go l.connectAndSubscribe()
	})
}

func (l *ControllerListener) connectAndSubscribe() {
	retries := getEnvInt("LED_MQTT_CONNECT_RETRIES", 10)
	if retries < 0 {
		retries = 0
	}
	delayMS := getEnvInt("LED_MQTT_CONNECT_RETRY_DELAY_MS", 2000)
	if delayMS < 100 {
		delayMS = 100
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(delayMS) * time.Millisecond)
		}
		if err := l.connect(); err != nil {
			lastErr = err
			log.Printf("[LED] Controller listener connect attempt %d/%d failed: %v", attempt+1, retries+1, err)
			continue
		}
		lastErr = nil
		break
	}

	if lastErr != nil {
		log.Printf("[LED] Controller listener failed to connect after %d attempts: %v", retries+1, lastErr)
		l.dryRun = true
	}
}

func (l *ControllerListener) connect() error {
	opts := mqtt.NewClientOptions()
	scheme := "tcp"
	if l.config.UseTLS {
		scheme = "ssl"
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: false})
	}
	brokerURL := fmt.Sprintf("%s://%s:%d", scheme, l.config.Host, l.config.Port)

	opts.AddBroker(brokerURL)
	opts.SetClientID(l.config.ClientID)
	if l.config.Username != "" {
		opts.SetUsername(l.config.Username)
		opts.SetPassword(l.config.Password)
	}

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Printf("[LED] Controller listener connection lost: %v", err)
	})

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("[LED] Controller listener connected to %s", brokerURL)
		l.subscribe(c)
	})

	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	l.client = mqtt.NewClient(opts)
	token := l.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (l *ControllerListener) subscribe(client mqtt.Client) {
	if client == nil {
		return
	}
	token := client.Subscribe(l.topicFilter, 1, l.handleStatusMessage)
	if token.Wait() && token.Error() != nil {
		log.Printf("[LED] Controller listener subscribe failed: %v", token.Error())
		return
	}
	log.Printf("[LED] Controller listener subscribed to %s", l.topicFilter)
}

func (l *ControllerListener) handleStatusMessage(_ mqtt.Client, msg mqtt.Message) {
	if len(msg.Payload()) == 0 {
		return
	}

	var payload models.LEDControllerHeartbeat
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("[LED] Controller listener received invalid JSON on %s: %v", msg.Topic(), err)
		return
	}

	topicSuffix := l.extractTopicSuffix(msg.Topic())

	identifier := strings.TrimSpace(payload.ControllerID)
	if derived := deriveControllerIdentifier(identifier, payload.TopicSuffix, topicSuffix, payload.MacAddress); derived != "" {
		identifier = derived
		payload.ControllerID = derived
	}

	if identifier == "" {
		identifier = l.extractIdentifier(msg.Topic())
	}
	if identifier == "" {
		log.Printf("[LED] Controller listener heartbeat missing controller_id (topic: %s)", msg.Topic())
		return
	}

	if payload.TopicSuffix == "" {
		payload.TopicSuffix = topicSuffix
	}

	if controllerHeartbeatHandler == nil {
		log.Printf("[LED] Controller listener received heartbeat for %s but no handler is registered", identifier)
		return
	}

	if err := controllerHeartbeatHandler(identifier, &payload); err != nil {
		log.Printf("[LED] Controller listener failed to store heartbeat for %s: %v", identifier, err)
		return
	}

	log.Printf("[LED] Controller heartbeat processed for %s (topic %s)", identifier, msg.Topic())
}

func (l *ControllerListener) extractIdentifier(topic string) string {
	return l.extractTopicSuffix(topic)
}

func (l *ControllerListener) extractTopicSuffix(topic string) string {
	topic = strings.Trim(topic, "/")
	if topic == "" {
		return ""
	}
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return ""
}

// Close disconnects the listener.
func (l *ControllerListener) Close() {
	if l.client != nil && l.client.IsConnected() {
		l.client.Disconnect(250)
		log.Println("[LED] Controller listener disconnected")
	}
}

func deriveControllerIdentifier(currentID, payloadTopicSuffix, topicSuffix, mac string) string {
	macSuffix := macSuffixFromAddress(mac)
	if macSuffix == "" {
		return ""
	}

	if !shouldAutogenerateControllerID(currentID) &&
		!shouldAutogenerateControllerID(payloadTopicSuffix) &&
		!shouldAutogenerateControllerID(topicSuffix) {
		return ""
	}

	prefix := resolveControllerPrefix(currentID, payloadTopicSuffix, topicSuffix)
	if prefix == "" {
		prefix = "esp"
	}

	newID := fmt.Sprintf("%s-%s", prefix, macSuffix)
	if strings.EqualFold(strings.TrimSpace(currentID), newID) {
		return ""
	}
	return newID
}

func shouldAutogenerateControllerID(id string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(id))
	if trimmed == "" {
		return true
	}
	if strings.HasSuffix(trimmed, "-000000") {
		return true
	}
	return false
}

func resolveControllerPrefix(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			continue
		}

		if idx := strings.Index(trimmed, "-"); idx > 0 {
			return strings.Trim(trimmed[:idx], "-")
		}

		if trimmed != "" && trimmed != "000000" {
			return strings.Trim(trimmed, "-")
		}
	}
	return ""
}

func macSuffixFromAddress(mac string) string {
	clean := strings.TrimSpace(strings.ToLower(mac))
	if clean == "" {
		return ""
	}

	replacer := strings.NewReplacer(":", "", "-", "", ".", "", " ", "")
	clean = replacer.Replace(clean)
	if len(clean) < 6 {
		return ""
	}

	for _, r := range clean {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return ""
		}
	}

	return clean[len(clean)-6:]
}
