package stellar

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
	rpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
)

type RPCClient struct {
	client *rpcclient.Client
}

func NewRPCClient(rpcURL string) Client {
	return &RPCClient{
		client: rpcclient.NewClient(rpcURL, nil),
	}
}

func (c *RPCClient) LatestLedger(ctx context.Context) (uint32, error) {
	ledger, err := c.client.GetLatestLedger(ctx)
	if err != nil {
		return 0, fmt.Errorf("get latest ledger: %w", err)
	}

	return ledger.Sequence, nil
}

func (c *RPCClient) FetchEvents(
	ctx context.Context,
	network string,
	input FetchEventsInput,
) ([]ContractEvent, uint32, error) {
	request := rpc.GetEventsRequest{
		StartLedger: input.StartLedger,
		Pagination: &rpc.PaginationOptions{
			Limit: uint(input.Limit),
		},
		Filters: buildEventFilters(input.ContractIDs),
	}

	response, err := c.client.GetEvents(ctx, request)
	if err != nil {
		return nil, 0, fmt.Errorf("get events: %w", err)
	}

	events := make([]ContractEvent, 0, len(response.Events))
	for _, event := range response.Events {
		events = append(events, mapEvent(network, event))
	}

	nextLedger := input.StartLedger
	if response.Cursor != "" {
		if parsed, err := parseCursorLedger(response.Cursor); err == nil && parsed > nextLedger {
			nextLedger = parsed
		}
	}
	if response.LatestLedger > nextLedger {
		nextLedger = response.LatestLedger
	}

	return events, nextLedger, nil
}

func buildEventFilters(contractIDs []string) []rpc.EventFilter {
	if len(contractIDs) == 0 {
		return []rpc.EventFilter{{
			EventType: rpc.EventTypeSet{
				rpc.EventTypeContract: nil,
			},
		}}
	}

	filters := make([]rpc.EventFilter, 0, len(contractIDs))
	for _, contractID := range contractIDs {
		filters = append(filters, rpc.EventFilter{
			EventType: rpc.EventTypeSet{
				rpc.EventTypeContract: nil,
			},
			ContractIDs: []string{contractID},
		})
	}

	return filters
}

func mapEvent(network string, event rpc.EventInfo) ContractEvent {
	return ContractEvent{
		ID:         event.ID,
		Network:    network,
		ContractID: event.ContractID,
		Ledger:     uint32(event.Ledger),
		TxnHash:    event.TransactionHash,
		EventType:  eventTypeToUint32(event.EventType),
		TopicsXDR:  append([]string(nil), event.TopicXDR...),
		ValueXDR:   event.ValueXDR,
		TopicsJSON: append([]json.RawMessage(nil), event.TopicJSON...),
		ValueJSON:  event.ValueJSON,
	}
}

func eventTypeToUint32(eventType string) uint32 {
	switch eventType {
	case rpc.EventTypeContract:
		return 1
	case rpc.EventTypeSystem:
		return 0
	case rpc.EventTypeDiagnostic:
		return 2
	default:
		return 0
	}
}

func parseCursorLedger(cursor string) (uint32, error) {
	parts := strings.Split(cursor, ".")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid cursor")
	}

	var ledger uint32
	if _, err := fmt.Sscanf(parts[0], "%d", &ledger); err != nil {
		return 0, fmt.Errorf("parse cursor ledger: %w", err)
	}

	return ledger, nil
}
