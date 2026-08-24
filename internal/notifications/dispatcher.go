package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

type Dispatcher struct {
	store  *storage.Store
	policy TargetPolicy
	log    *slog.Logger
}

func NewDispatcher(store *storage.Store, policy TargetPolicy, log *slog.Logger) *Dispatcher {
	return &Dispatcher{store: store, policy: policy, log: log}
}

func (d *Dispatcher) DispatchPending(ctx context.Context) {
	items, err := d.store.PendingFindingNotifications(ctx, 50)
	if err != nil {
		d.log.Warn("load pending finding notifications", "error", err)
		return
	}
	for _, item := range items {
		destination, err := d.store.GetNotificationDestination(ctx, item.DestinationID, true)
		if err == nil {
			err = d.send(ctx, destination, item)
		}
		if recordErr := d.store.RecordFindingNotification(ctx, item.EventID, item.DestinationID, err); recordErr != nil {
			d.log.Warn("record finding notification", "error", recordErr)
		}
		if err != nil {
			d.log.Warn("deliver finding notification", "destination_id", item.DestinationID, "event_type", item.EventType, "error", err)
		}
	}
}

func (d *Dispatcher) send(ctx context.Context, destination models.NotificationDestination, delivery models.FindingNotificationDelivery) error {
	var provider Provider
	switch destination.Provider {
	case "ntfy":
		n := NewNtfy(destination.Config["serverUrl"], destination.Config["topic"], d.policy)
		n.Token, n.Username, n.Password = destination.Config["token"], destination.Config["username"], destination.Config["password"]
		provider = n
	case "webhook":
		hook, err := NewWebhook(destination.Config["webhookUrl"], nil, d.policy)
		if err != nil {
			return err
		}
		provider = hook
	default:
		return fmt.Errorf("unsupported notification provider %q", destination.Provider)
	}
	finding := delivery.Finding
	serverName := finding.ServerID
	if server, err := d.store.GetServer(ctx, finding.ServerID, false); err == nil {
		serverName = server.Name
	}
	lines := []string{fmt.Sprintf("Event: %s", strings.ReplaceAll(delivery.EventType, "_", " ")), "Server: " + serverName, "Category: " + finding.Category, "Severity: " + string(finding.Severity), finding.Summary}
	if len(finding.Evidence) > 0 {
		lines = append(lines, fmt.Sprintf("Evidence: %s = %s", finding.Evidence[0].Label, finding.Evidence[0].Value))
	}
	if len(finding.Suggestions) > 0 {
		lines = append(lines, "Investigate: "+finding.Suggestions[0].Title)
	}
	lines = append(lines, "PGSentinel finding: "+finding.Fingerprint, "Observed: "+finding.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	return provider.Send(ctx, Message{Title: "PGSentinel: " + finding.Title, Body: strings.Join(lines, "\n"), Severity: string(finding.Severity)})
}
