package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stellar/go-stellar-sdk/keypair"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/repository"
)

type mockProjectRepo struct {
	project model.SchemaProject
}

func (m *mockProjectRepo) Insert(context.Context, string, string, string, string, string, string, string) (model.SchemaProject, error) {
	return model.SchemaProject{}, nil
}
func (m *mockProjectRepo) ListByUser(context.Context, string) ([]model.SchemaProject, error) {
	return nil, nil
}
func (m *mockProjectRepo) GetByIDForUser(_ context.Context, projectID, userID string) (model.SchemaProject, error) {
	if m.project.ID == projectID && m.project.UserID == userID {
		return m.project, nil
	}
	return model.SchemaProject{}, repository.ErrSchemaProjectNotFound
}
func (m *mockProjectRepo) Update(context.Context, string, string, string, string, string) (model.SchemaProject, error) {
	return model.SchemaProject{}, nil
}
func (m *mockProjectRepo) MarkPublishedByPublishTokenID(context.Context, string, string, string) error {
	return nil
}
func (m *mockProjectRepo) SyncPublishedStatus(context.Context, string, string) (model.SchemaProject, error) {
	return model.SchemaProject{}, nil
}
func (m *mockProjectRepo) SlugExists(context.Context, string, string) (bool, error) {
	return false, nil
}

type mockVerifySchemaRepo struct {
	items []model.EventSchema
}

func (m *mockVerifySchemaRepo) ListPublishedByPublisher(_ context.Context, publisherUserID string) ([]model.EventSchema, error) {
	var out []model.EventSchema
	for _, item := range m.items {
		if item.PublisherUserID != nil && *item.PublisherUserID == publisherUserID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *mockVerifySchemaRepo) VerifyContractForPublisher(
	_ context.Context,
	publisherUserID, contractID, network, wallet string,
	verifiedAt time.Time,
) ([]model.EventSchema, error) {
	events := map[string]model.EventSchema{}
	for _, item := range m.items {
		if item.PublisherUserID == nil || *item.PublisherUserID != publisherUserID {
			continue
		}
		if item.ContractID != contractID || item.Network != network {
			continue
		}
		if item.TrustTier != "community" || item.Status != "published" {
			continue
		}
		current, ok := events[item.EventName]
		if !ok || item.Version > current.Version {
			events[item.EventName] = item
		}
	}
	if len(events) == 0 {
		return nil, repository.ErrSchemaNotFound
	}

	var promoted []model.EventSchema
	for eventName, latest := range events {
		for i, item := range m.items {
			if item.ContractID == contractID && item.Network == network && item.EventName == eventName &&
				item.TrustTier == "verified" && item.Status == "published" {
				m.items[i].Status = "archived"
			}
		}
		for i, item := range m.items {
			if item.ID == latest.ID {
				m.items[i].TrustTier = "verified"
				m.items[i].VerifiedWallet = &wallet
				m.items[i].VerifiedAt = &verifiedAt
				promoted = append(promoted, m.items[i])
			}
		}
	}
	return promoted, nil
}

type mockChallengeRepo struct {
	message string
	used    bool
}

func (m *mockChallengeRepo) Create(_ context.Context, _, _, _, _, nonce, message string, _ time.Time) error {
	m.message = message
	return nil
}

func (m *mockChallengeRepo) Consume(_ context.Context, _, _, _, _, nonce string, _ time.Time) (string, error) {
	if m.used {
		return "", repository.ErrVerifyChallengeUsed
	}
	if m.message == "" {
		return "", repository.ErrVerifyChallengeNotFound
	}
	m.used = true
	return m.message, nil
}

type mockAuthorityChecker struct {
	allowed bool
}

func (m *mockAuthorityChecker) IsAuthorizedWallet(context.Context, string, string, string) (bool, error) {
	return m.allowed, nil
}

type mockAuthParser struct {
	userID string
}

func (m *mockAuthParser) ParseUserID(string) (string, error) {
	return m.userID, nil
}

func signChallengeMessage(kp *keypair.Full, message string) string {
	payload := append([]byte(stellarSignedMessagePrefix), []byte(message)...)
	digest := sha256.Sum256(payload)
	sig, err := kp.Sign(digest[:])
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func TestVerifyContractPromotesAllEvents(t *testing.T) {
	kp, err := keypair.Random()
	if err != nil {
		t.Fatal(err)
	}

	publisher := "user-1"
	projectID := "project-1"
	contractID := "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	network := "testnet"

	repo := &mockVerifySchemaRepo{
		items: []model.EventSchema{
			{ID: "s1", ContractID: contractID, Network: network, EventName: "transfer", Version: 1, PublisherUserID: &publisher, TrustTier: "community", Status: "published"},
			{ID: "s2", ContractID: contractID, Network: network, EventName: "mint", Version: 1, PublisherUserID: &publisher, TrustTier: "community", Status: "published"},
		},
	}

	chRepo := &mockChallengeRepo{}
	svc := NewVerifyService(
		&mockProjectRepo{project: model.SchemaProject{ID: projectID, UserID: publisher}},
		repo,
		chRepo,
		&mockAuthorityChecker{allowed: true},
		&mockAuthParser{userID: publisher},
		network,
	)

	challenge, err := svc.CreateChallenge(context.Background(), "token", projectID, model.VerifyChallengeInput{
		ContractID: contractID,
		Network:    network,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.VerifyContract(context.Background(), "token", projectID, model.VerifyContractInput{
		ContractID: contractID,
		Network:    network,
		Wallet:     kp.Address(),
		Signature:  signChallengeMessage(kp, challenge.Message),
		Nonce:      challenge.Nonce,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 verified events, got %d", len(result.Events))
	}
}

func TestVerifyContractWrongWalletForbidden(t *testing.T) {
	kp, err := keypair.Random()
	if err != nil {
		t.Fatal(err)
	}

	publisher := "user-1"
	projectID := "project-1"
	contractID := "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	repo := &mockVerifySchemaRepo{
		items: []model.EventSchema{
			{ID: "s1", ContractID: contractID, Network: "testnet", EventName: "transfer", Version: 1, PublisherUserID: &publisher, TrustTier: "community", Status: "published"},
		},
	}

	svc := NewVerifyService(
		&mockProjectRepo{project: model.SchemaProject{ID: projectID, UserID: publisher}},
		repo,
		&mockChallengeRepo{message: "challenge"},
		&mockAuthorityChecker{allowed: false},
		&mockAuthParser{userID: publisher},
		"testnet",
	)

	_, err = svc.VerifyContract(context.Background(), "token", projectID, model.VerifyContractInput{
		ContractID: contractID,
		Network:    "testnet",
		Wallet:     kp.Address(),
		Signature:  signChallengeMessage(kp, "challenge"),
		Nonce:      "nonce",
	})
	if err != ErrContractAuthorityDenied {
		t.Fatalf("expected ErrContractAuthorityDenied, got %v", err)
	}
}

func TestVerifyContractBadSignature(t *testing.T) {
	publisher := "user-1"
	projectID := "project-1"
	contractID := "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	svc := NewVerifyService(
		&mockProjectRepo{project: model.SchemaProject{ID: projectID, UserID: publisher}},
		&mockVerifySchemaRepo{},
		&mockChallengeRepo{message: "challenge"},
		&mockAuthorityChecker{allowed: true},
		&mockAuthParser{userID: publisher},
		"testnet",
	)

	kp, _ := keypair.Random()
	_, err := svc.VerifyContract(context.Background(), "token", projectID, model.VerifyContractInput{
		ContractID: contractID,
		Network:    "testnet",
		Wallet:     kp.Address(),
		Signature:  base64.StdEncoding.EncodeToString([]byte("bad-signature")),
		Nonce:      "nonce",
	})
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestBuildContractPublicationsCanVerify(t *testing.T) {
	publisher := "user-1"
	contractID := "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	now := time.Now()

	publications := buildContractPublications([]model.EventSchema{
		{ContractID: contractID, Network: "testnet", EventName: "transfer", Version: 1, PublisherUserID: &publisher, TrustTier: "community", Status: "published", CreatedAt: now},
		{ContractID: contractID, Network: "testnet", EventName: "transfer", Version: 2, PublisherUserID: &publisher, TrustTier: "community", Status: "published", CreatedAt: now},
		{ContractID: contractID, Network: "testnet", EventName: "transfer", Version: 1, PublisherUserID: &publisher, TrustTier: "verified", Status: "published", CreatedAt: now},
	})

	if len(publications) != 1 {
		t.Fatalf("expected 1 contract group, got %d", len(publications))
	}
	if !publications[0].CanVerify {
		t.Fatal("expected canVerify true when newer community version exists")
	}
}
