package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"gofr.dev/pkg/gofr/datasource"
)

const (
	natsBackend            = "client"
	jetStreamStatusOK      = "OK"
	jetStreamStatusError   = "Error"
	jetStreamConnected     = "CONNECTED"
	jetStreamConnecting    = "CONNECTING"
	jetStreamDisconnecting = "DISCONNECTED"
	natsHealthCheckTimeout = 5 * time.Second
)

// Health returns the health status of the client client.
func (n *client) Health() datasource.Health {
	h := datasource.Health{
		Status:  datasource.StatusUp,
		Details: make(map[string]interface{}),
	}

	connectionStatus := n.Conn.Status()

	switch connectionStatus {
	case nats.CONNECTING:
		h.Status = datasource.StatusUp
		h.Details["connection_status"] = jetStreamConnecting

		n.Logger.Debug("client health check: Connecting")
	case nats.CONNECTED:
		h.Details["connection_status"] = jetStreamConnected

		n.Logger.Debug("client health check: Connected")
	case nats.CLOSED, nats.DISCONNECTED, nats.RECONNECTING, nats.DRAINING_PUBS, nats.DRAINING_SUBS:
		h.Status = datasource.StatusDown
		h.Details["connection_status"] = jetStreamDisconnecting

		n.Logger.Error("client health check: Disconnected")
	default:
		h.Status = datasource.StatusDown
		h.Details["connection_status"] = connectionStatus.String()

		n.Logger.Error("client health check: Unknown status", connectionStatus)
	}

	h.Details["host"] = n.Config.Server
	h.Details["backend"] = natsBackend
	h.Details["jetstream_enabled"] = n.JetStream != nil

	ctx, cancel := context.WithTimeout(context.Background(), natsHealthCheckTimeout)
	defer cancel()

	if n.JetStream != nil && connectionStatus == nats.CONNECTED {
		status := getJetStreamStatus(ctx, n.JetStream)

		h.Details["jetstream_status"] = status

		if status != jetStreamStatusOK {
			n.Logger.Error("client health check: JetStream error:", status)
		} else {
			n.Logger.Debug("client health check: JetStream enabled")
		}
	} else if n.JetStream == nil {
		n.Logger.Debug("client health check: JetStream not enabled")
	}

	return h
}

func getJetStreamStatus(ctx context.Context, js jetstream.JetStream) string {
	_, err := js.AccountInfo(ctx)
	if err != nil {
		return jetStreamStatusError + ": " + err.Error()
	}

	return jetStreamStatusOK
}
