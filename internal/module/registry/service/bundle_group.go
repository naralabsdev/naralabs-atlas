package service

import (
	"sort"
	"strings"
	"time"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

const publishBatchWindow = 30 * time.Second

type schemaBundle struct {
	Version    int
	TrustTier  string
	Author     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	VerifiedAt *time.Time
	Events     []model.EventSchema
}

func publisherKey(id *string) string {
	if id == nil {
		return ""
	}
	return strings.TrimSpace(*id)
}

func groupIntoBundles(schemas []model.EventSchema) []schemaBundle {
	published := make([]model.EventSchema, 0, len(schemas))
	for _, schema := range schemas {
		if schema.Status == "published" {
			published = append(published, schema)
		}
	}
	if len(published) == 0 {
		return nil
	}

	sort.Slice(published, func(i, j int) bool {
		if published[i].CreatedAt.Equal(published[j].CreatedAt) {
			return published[i].EventName < published[j].EventName
		}
		return published[i].CreatedAt.Before(published[j].CreatedAt)
	})

	batches := make([][]model.EventSchema, 0)
	current := []model.EventSchema{published[0]}
	anchor := published[0].CreatedAt
	publisher := publisherKey(published[0].PublisherUserID)

	for _, schema := range published[1:] {
		nextPublisher := publisherKey(schema.PublisherUserID)
		if nextPublisher != publisher || schema.CreatedAt.Sub(anchor) > publishBatchWindow {
			batches = append(batches, current)
			current = []model.EventSchema{schema}
			anchor = schema.CreatedAt
			publisher = nextPublisher
			continue
		}
		current = append(current, schema)
	}
	batches = append(batches, current)

	out := make([]schemaBundle, 0, len(batches))
	for _, batch := range batches {
		out = append(out, finalizeBundle(batch))
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Version == out[j].Version {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].Version > out[j].Version
	})

	return out
}

func finalizeBundle(batch []model.EventSchema) schemaBundle {
	bundle := schemaBundle{
		Events: append([]model.EventSchema(nil), batch...),
	}

	for _, schema := range batch {
		if schema.Version > bundle.Version {
			bundle.Version = schema.Version
		}
		if bundle.CreatedAt.IsZero() || schema.CreatedAt.Before(bundle.CreatedAt) {
			bundle.CreatedAt = schema.CreatedAt
		}
		if schema.UpdatedAt.After(bundle.UpdatedAt) {
			bundle.UpdatedAt = schema.UpdatedAt
		}
		if schema.Author != nil && strings.TrimSpace(*schema.Author) != "" && bundle.Author == "" {
			bundle.Author = strings.TrimSpace(*schema.Author)
		}
		if schema.TrustTier == "verified" {
			bundle.TrustTier = "verified"
			if schema.VerifiedAt != nil {
				if bundle.VerifiedAt == nil || schema.VerifiedAt.After(*bundle.VerifiedAt) {
					copyTime := *schema.VerifiedAt
					bundle.VerifiedAt = &copyTime
				}
			}
		}
	}

	if bundle.TrustTier == "" {
		bundle.TrustTier = "community"
	}

	return bundle
}

func contractVerification(schemas []model.EventSchema) (bool, *time.Time) {
	var verifiedAt *time.Time
	isVerified := false

	for _, schema := range schemas {
		if schema.Status != "published" || schema.TrustTier != "verified" {
			continue
		}
		isVerified = true
		if schema.VerifiedAt == nil {
			continue
		}
		if verifiedAt == nil || schema.VerifiedAt.After(*verifiedAt) {
			copyTime := *schema.VerifiedAt
			verifiedAt = &copyTime
		}
	}

	return isVerified, verifiedAt
}

func toBundleSummary(bundle schemaBundle) model.SchemaBundleSummary {
	return model.SchemaBundleSummary{
		Version:    bundle.Version,
		TrustTier:  bundle.TrustTier,
		EventCount: len(bundle.Events),
		Author:     bundle.Author,
		UpdatedAt:  bundle.UpdatedAt,
		VerifiedAt: bundle.VerifiedAt,
	}
}

func toBundleSummaries(bundles []schemaBundle) []model.SchemaBundleSummary {
	out := make([]model.SchemaBundleSummary, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, toBundleSummary(bundle))
	}
	return out
}

func findBundleByVersion(bundles []schemaBundle, version int) (*schemaBundle, bool) {
	for i := range bundles {
		if bundles[i].Version == version {
			return &bundles[i], true
		}
	}
	return nil, false
}
