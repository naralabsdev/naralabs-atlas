package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	registermodel "github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	registryservice "github.com/naralabs/naralabs-atlas/internal/module/registry/service"
	"github.com/naralabs/naralabs-atlas/lib/decoder"
	"github.com/naralabs/naralabs-atlas/lib/scval"
)

const maxBatchSize = 50

var ErrInvalidDecodeInput = errors.New("INVALID_DECODE_INPUT")
var ErrBatchTooLarge = errors.New("BATCH_TOO_LARGE")

type SchemaLookup interface {
	ListByContract(ctx context.Context, network, contractID string) ([]registermodel.EventSchema, error)
	GetEventSchema(ctx context.Context, network, contractID, eventName string, version int) (registermodel.EventSchema, error)
}

type DecodeService struct {
	schemas SchemaLookup
	parser  *scval.Parser
}

func NewDecodeService(schemas SchemaLookup) *DecodeService {
	return &DecodeService{
		schemas: schemas,
		parser:  scval.DefaultParser,
	}
}

type DecodeEventInput struct {
	Network       string
	ContractID    string
	EventName     string
	SchemaVersion int
	TopicsXDR     []string
	TopicsJSON    []json.RawMessage
	ValueXDR      string
	ValueJSON     json.RawMessage
}

type DecodeEventResult struct {
	DecodeStatus  string         `json:"decodeStatus"`
	EventName     string         `json:"eventName,omitempty"`
	SchemaVersion int            `json:"schemaVersion,omitempty"`
	Fields        map[string]any `json:"fields,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	Level2        decoder.Level2Payload `json:"level2"`
}

type BatchDecodeResult struct {
	Items []DecodeEventResult `json:"items"`
	Total int                 `json:"total"`
}

func (s *DecodeService) Decode(ctx context.Context, input DecodeEventInput) (DecodeEventResult, error) {
	if err := validateDecodeInput(input); err != nil {
		return DecodeEventResult{}, err
	}

	topics, value, err := s.normalizePayload(input)
	if err != nil {
		return DecodeEventResult{}, err
	}

	schema, found, err := s.resolveSchema(ctx, input)
	if err != nil {
		return DecodeEventResult{}, err
	}
	if !found {
		raw := decoder.DecodeEvent(topics, value, nil, input.EventName)
		return toServiceResult(raw), nil
	}

	decoded := decoder.DecodeEventWithVersion(
		topics,
		value,
		schema.SchemaBody,
		input.EventName,
		schema.Version,
	)
	return toServiceResult(decoded), nil
}

func (s *DecodeService) DecodeBatch(ctx context.Context, items []DecodeEventInput) (BatchDecodeResult, error) {
	if len(items) == 0 {
		return BatchDecodeResult{}, ErrInvalidDecodeInput
	}
	if len(items) > maxBatchSize {
		return BatchDecodeResult{}, ErrBatchTooLarge
	}

	out := BatchDecodeResult{
		Items: make([]DecodeEventResult, 0, len(items)),
		Total: len(items),
	}
	for _, item := range items {
		result, err := s.Decode(ctx, item)
		if err != nil {
			return BatchDecodeResult{}, err
		}
		out.Items = append(out.Items, result)
	}
	return out, nil
}

func (s *DecodeService) normalizePayload(input DecodeEventInput) ([]json.RawMessage, json.RawMessage, error) {
	topicsRaw := s.parser.NormalizeTopics(input.TopicsXDR, input.TopicsJSON)
	valueRaw := s.parser.NormalizeValue(input.ValueXDR, input.ValueJSON)

	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		return nil, nil, fmt.Errorf("parse topics: %w", err)
	}
	return topics, valueRaw, nil
}

func (s *DecodeService) resolveSchema(
	ctx context.Context,
	input DecodeEventInput,
) (registermodel.EventSchema, bool, error) {
	network := strings.ToLower(strings.TrimSpace(input.Network))
	contractID := strings.TrimSpace(input.ContractID)
	eventName := strings.TrimSpace(input.EventName)

	if eventName != "" {
		schema, err := s.schemas.GetEventSchema(ctx, network, contractID, eventName, input.SchemaVersion)
		if errors.Is(err, registryservice.ErrSchemaNotFound) {
			return registermodel.EventSchema{}, false, nil
		}
		if err != nil {
			return registermodel.EventSchema{}, false, err
		}
		return schema, true, nil
	}

	schemas, err := s.schemas.ListByContract(ctx, network, contractID)
	if err != nil {
		return registermodel.EventSchema{}, false, err
	}

	topics, value, err := s.normalizePayload(input)
	if err != nil {
		return registermodel.EventSchema{}, false, err
	}

	for _, schema := range schemas {
		if schema.Status != "" && schema.Status != "published" {
			continue
		}
		preview := decoder.DecodeEvent(topics, value, schema.SchemaBody, schema.EventName)
		if preview.DecodeStatus == "decoded" {
			return schema, true, nil
		}
	}
	return registermodel.EventSchema{}, false, nil
}

func validateDecodeInput(input DecodeEventInput) error {
	if strings.TrimSpace(input.Network) == "" {
		return ErrInvalidDecodeInput
	}
	if strings.TrimSpace(input.ContractID) == "" {
		return ErrInvalidDecodeInput
	}
	if len(input.TopicsXDR) == 0 && len(input.TopicsJSON) == 0 {
		return ErrInvalidDecodeInput
	}
	if strings.TrimSpace(input.ValueXDR) == "" && len(input.ValueJSON) == 0 {
		return ErrInvalidDecodeInput
	}
	return nil
}

func toServiceResult(result decoder.Result) DecodeEventResult {
	return DecodeEventResult{
		DecodeStatus:  result.DecodeStatus,
		EventName:     result.EventName,
		SchemaVersion: result.SchemaVersion,
		Fields:        result.Fields,
		Summary:       result.Summary,
		Level2:        result.Level2,
	}
}
