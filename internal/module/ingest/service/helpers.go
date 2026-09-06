package service

import (
	"time"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
	"github.com/naralabs/naralabs-atlas/lib/scval/address"
	"github.com/naralabs/naralabs-atlas/lib/scval/token"
)

func ExtractDerived(events []model.ContractEvent) ([]model.EventAddress, []model.TokenEvent) {
	addresses := make([]model.EventAddress, 0, len(events))
	tokens := make([]model.TokenEvent, 0, len(events))

	for _, event := range events {
		for _, row := range address.Extract(event.TopicsJSON, event.ValueJSON) {
			addresses = append(addresses, model.EventAddress{
				Network:    event.Network,
				Address:    row.Address,
				ContractID: event.ContractID,
				EventID:    event.ID,
				Ledger:     event.Ledger,
				Role:       row.Role,
				IngestedAt: event.IngestedAt,
			})
		}

		if hint, ok := token.ExtractTransfer(event.TopicsJSON, event.ValueJSON); ok {
			tokens = append(tokens, model.TokenEvent{
				Network:     event.Network,
				ContractID:  event.ContractID,
				EventID:     event.ID,
				Ledger:      event.Ledger,
				TokenSymbol: hint.Symbol,
				TokenAmount: hint.Amount,
				FromAddress: hint.From,
				ToAddress:   hint.To,
				Action:      hint.Action,
				IngestedAt:  event.IngestedAt,
			})
		}
	}

	if len(addresses) == 0 && len(tokens) == 0 {
		return nil, nil
	}
	return addresses, tokens
}

func ResolveColdStart(lastLedger, latestLedger, retentionLedgers uint32, startLedgerRaw string) (uint32, error) {
	if lastLedger > 0 {
		return lastLedger, nil
	}
	if startLedgerRaw != "" {
		return config.ParseStartLedger(startLedgerRaw, latestLedger)
	}
	if latestLedger <= retentionLedgers {
		return 0, nil
	}
	return latestLedger - retentionLedgers, nil
}

func ReorgFromLedger(lastLedger, reorgWindow uint32) uint32 {
	if reorgWindow == 0 || lastLedger <= reorgWindow {
		return 0
	}
	return lastLedger - reorgWindow
}

func NextAdaptivePoll(current, min, max time.Duration, caughtUp bool) time.Duration {
	if min <= 0 {
		min = time.Second
	}
	if max <= 0 {
		max = current
	}
	if current <= 0 {
		current = min
	}
	if caughtUp {
		next := current + current/2
		if next > max {
			return max
		}
		return next
	}
	next := current / 2
	if next < min {
		return min
	}
	return next
}
