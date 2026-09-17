package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stellar/go-stellar-sdk/strkey"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/repository"
)

const stellarSignedMessagePrefix = "Stellar Signed Message:\n"

type VerifyChallengeRepository interface {
	Create(
		ctx context.Context,
		projectID, userID, contractID, network, nonce, message string,
		expiresAt time.Time,
	) error
	Consume(
		ctx context.Context,
		projectID, userID, contractID, network, nonce string,
		now time.Time,
	) (string, error)
}

type ContractAuthorityChecker interface {
	IsAuthorizedWallet(ctx context.Context, contractID, network, wallet string) (bool, error)
}

type VerifySchemaRepository interface {
	ListPublishedByPublisher(ctx context.Context, publisherUserID string) ([]model.EventSchema, error)
	VerifyContractForPublisher(
		ctx context.Context,
		publisherUserID, contractID, network, wallet string,
		verifiedAt time.Time,
	) ([]model.EventSchema, error)
}

type VerifyService struct {
	projects    ProjectRepository
	schemas     VerifySchemaRepository
	challenges  VerifyChallengeRepository
	authorities ContractAuthorityChecker
	auth        AuthUserParser
	defaultNet  string
	now         func() time.Time
}

func NewVerifyService(
	projects ProjectRepository,
	schemas VerifySchemaRepository,
	challenges VerifyChallengeRepository,
	authorities ContractAuthorityChecker,
	auth AuthUserParser,
	defaultNetwork string,
) *VerifyService {
	return &VerifyService{
		projects:    projects,
		schemas:     schemas,
		challenges:  challenges,
		authorities: authorities,
		auth:        auth,
		defaultNet:  strings.ToLower(strings.TrimSpace(defaultNetwork)),
		now:         time.Now,
	}
}

func (s *VerifyService) ListPublications(
	ctx context.Context,
	authorization, projectID string,
) (model.ProjectPublicationsResponse, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.ProjectPublicationsResponse{}, ErrUnauthorized
	}

	if _, err := s.projects.GetByIDForUser(ctx, projectID, userID); errors.Is(err, repository.ErrSchemaProjectNotFound) {
		return model.ProjectPublicationsResponse{}, ErrProjectNotFound
	} else if err != nil {
		return model.ProjectPublicationsResponse{}, err
	}

	schemas, err := s.schemas.ListPublishedByPublisher(ctx, userID)
	if err != nil {
		return model.ProjectPublicationsResponse{}, err
	}

	return model.ProjectPublicationsResponse{
		Contracts: buildContractPublications(schemas),
	}, nil
}

func (s *VerifyService) CreateChallenge(
	ctx context.Context,
	authorization, projectID string,
	input model.VerifyChallengeInput,
) (model.VerifyChallengeResponse, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.VerifyChallengeResponse{}, ErrUnauthorized
	}

	if _, err := s.projects.GetByIDForUser(ctx, projectID, userID); errors.Is(err, repository.ErrSchemaProjectNotFound) {
		return model.VerifyChallengeResponse{}, ErrProjectNotFound
	} else if err != nil {
		return model.VerifyChallengeResponse{}, err
	}

	contractID, err := normalizeContractID(input.ContractID)
	if err != nil {
		return model.VerifyChallengeResponse{}, err
	}

	network, err := s.resolveNetwork(input.Network)
	if err != nil {
		return model.VerifyChallengeResponse{}, err
	}

	nonce := uuid.NewString()
	expiresAt := s.now().Add(10 * time.Minute)
	message := fmt.Sprintf(
		"NaraLabs Schema Registry Verify\nContract: %s\nNetwork: %s\nNonce: %s\nExpires: %s",
		contractID,
		network,
		nonce,
		expiresAt.UTC().Format(time.RFC3339),
	)

	if err := s.challenges.Create(ctx, projectID, userID, contractID, network, nonce, message, expiresAt); err != nil {
		return model.VerifyChallengeResponse{}, err
	}

	return model.VerifyChallengeResponse{
		Nonce:     nonce,
		Message:   message,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *VerifyService) VerifyContract(
	ctx context.Context,
	authorization, projectID string,
	input model.VerifyContractInput,
) (model.VerifyContractResponse, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.VerifyContractResponse{}, ErrUnauthorized
	}

	if _, err := s.projects.GetByIDForUser(ctx, projectID, userID); errors.Is(err, repository.ErrSchemaProjectNotFound) {
		return model.VerifyContractResponse{}, ErrProjectNotFound
	} else if err != nil {
		return model.VerifyContractResponse{}, err
	}

	contractID, err := normalizeContractID(input.ContractID)
	if err != nil {
		return model.VerifyContractResponse{}, err
	}

	network, err := s.resolveNetwork(input.Network)
	if err != nil {
		return model.VerifyContractResponse{}, err
	}

	wallet := strings.TrimSpace(input.Wallet)
	if wallet == "" || !strings.HasPrefix(wallet, "G") {
		return model.VerifyContractResponse{}, ErrInvalidSchemaBody
	}

	nonce := strings.TrimSpace(input.Nonce)
	if nonce == "" {
		return model.VerifyContractResponse{}, ErrChallengeNotFound
	}

	now := s.now()
	message, err := s.challenges.Consume(ctx, projectID, userID, contractID, network, nonce, now)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrVerifyChallengeNotFound):
			return model.VerifyContractResponse{}, ErrChallengeNotFound
		case errors.Is(err, repository.ErrVerifyChallengeExpired):
			return model.VerifyContractResponse{}, ErrChallengeExpired
		case errors.Is(err, repository.ErrVerifyChallengeUsed):
			return model.VerifyContractResponse{}, ErrChallengeAlreadyUsed
		default:
			return model.VerifyContractResponse{}, err
		}
	}

	if err := verifyStellarSignedMessage(wallet, message, input.Signature); err != nil {
		return model.VerifyContractResponse{}, err
	}

	authorized, err := s.authorities.IsAuthorizedWallet(ctx, contractID, network, wallet)
	if err != nil {
		if errors.Is(err, ErrContractAuthorityUnavailable) {
			return model.VerifyContractResponse{}, ErrContractAuthorityUnavailable
		}
		return model.VerifyContractResponse{}, err
	}
	if !authorized {
		return model.VerifyContractResponse{}, ErrContractAuthorityDenied
	}

	promoted, err := s.schemas.VerifyContractForPublisher(ctx, userID, contractID, network, wallet, now)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			return model.VerifyContractResponse{}, ErrNothingToVerify
		}
		return model.VerifyContractResponse{}, err
	}

	response := model.VerifyContractResponse{
		ContractID: contractID,
		Network:    network,
		Wallet:     wallet,
		VerifiedAt: now,
		Events:     make([]model.VerifiedEventResult, 0, len(promoted)),
	}
	for _, schema := range promoted {
		response.Events = append(response.Events, model.VerifiedEventResult{
			EventName: schema.EventName,
			SchemaID:  schema.ID,
			Version:   schema.Version,
		})
	}
	return response, nil
}

func (s *VerifyService) resolveNetwork(network string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(network))
	if n == "" {
		if s.defaultNet == "" {
			return "", ErrInvalidNetwork
		}
		return s.defaultNet, nil
	}
	return normalizeNetwork(n)
}

func verifyStellarSignedMessage(wallet, message, signatureB64 string) error {
	signatureRaw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureB64))
	if err != nil {
		return ErrInvalidSignature
	}

	pubKey, err := strkey.Decode(strkey.VersionByteAccountID, wallet)
	if err != nil {
		return ErrInvalidSignature
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return ErrInvalidSignature
	}

	payload := append([]byte(stellarSignedMessagePrefix), []byte(message)...)
	digest := sha256.Sum256(payload)
	if !ed25519.Verify(ed25519.PublicKey(pubKey), digest[:], signatureRaw) {
		return ErrInvalidSignature
	}
	return nil
}

func buildContractPublications(schemas []model.EventSchema) []model.ContractPublication {
	type contractKey struct {
		contractID string
		network    string
	}

	eventsByContract := make(map[contractKey]map[string][]model.EventSchema)
	contractOrder := make([]contractKey, 0)

	for _, schema := range schemas {
		key := contractKey{contractID: schema.ContractID, network: schema.Network}
		if _, ok := eventsByContract[key]; !ok {
			eventsByContract[key] = make(map[string][]model.EventSchema)
			contractOrder = append(contractOrder, key)
		}
		eventsByContract[key][schema.EventName] = append(eventsByContract[key][schema.EventName], schema)
	}

	out := make([]model.ContractPublication, 0, len(contractOrder))
	for _, key := range contractOrder {
		eventMap := eventsByContract[key]
		eventNames := make([]string, 0, len(eventMap))
		for eventName := range eventMap {
			eventNames = append(eventNames, eventName)
		}
		// stable order
		sort.Strings(eventNames)

		publication := model.ContractPublication{
			ContractID: key.contractID,
			Network:    key.network,
			Events:     make([]model.EventPublication, 0, len(eventNames)),
		}

		canVerify := false
		for _, eventName := range eventNames {
			versions := eventMap[eventName]
			eventPub := model.EventPublication{
				EventName: eventName,
				Versions:  make([]model.SchemaVersionSummary, 0, len(versions)),
			}

			var latestCommunity *model.SchemaVersionSummary
			var verified *model.VerifiedSummary

			for _, version := range versions {
				summary := model.SchemaVersionSummary{
					ID:        version.ID,
					Version:   version.Version,
					TrustTier: version.TrustTier,
					Status:    version.Status,
					CreatedAt: version.CreatedAt,
				}
				eventPub.Versions = append(eventPub.Versions, summary)

				if version.TrustTier == "community" && version.Status == "published" {
					if latestCommunity == nil || version.Version > latestCommunity.Version {
						copySummary := summary
						latestCommunity = &copySummary
					}
				}
				if version.TrustTier == "verified" && version.Status == "published" {
					if verified == nil || version.Version > verified.Version {
						verified = &model.VerifiedSummary{
							ID:      version.ID,
							Version: version.Version,
						}
						if version.VerifiedWallet != nil {
							verified.VerifiedWallet = *version.VerifiedWallet
						}
						if version.VerifiedAt != nil {
							verified.VerifiedAt = *version.VerifiedAt
						}
					}
				}
			}

			eventPub.LatestCommunity = latestCommunity
			eventPub.Verified = verified

			if latestCommunity != nil {
				if verified == nil || verified.Version < latestCommunity.Version {
					canVerify = true
				}
			}

			publication.Events = append(publication.Events, eventPub)
		}

		publication.CanVerify = canVerify
		out = append(out, publication)
	}

	return out
}
