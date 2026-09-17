package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ContractAuthorityRepository interface {
	Get(ctx context.Context, contractID, network string) (string, bool, error)
	Upsert(ctx context.Context, contractID, network, wallet string) error
}

type ContractAuthorityService struct {
	repo       ContractAuthorityRepository
	httpClient *http.Client
}

func NewContractAuthorityService(repo ContractAuthorityRepository) *ContractAuthorityService {
	return &ContractAuthorityService{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *ContractAuthorityService) IsAuthorizedWallet(
	ctx context.Context,
	contractID, network, wallet string,
) (bool, error) {
	deployer, err := s.ResolveDeployer(ctx, contractID, network)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(deployer, strings.TrimSpace(wallet)), nil
}

func (s *ContractAuthorityService) ResolveDeployer(
	ctx context.Context,
	contractID, network string,
) (string, error) {
	if cached, ok, err := s.repo.Get(ctx, contractID, network); err != nil {
		return "", err
	} else if ok {
		return cached, nil
	}

	deployer, err := s.lookupDeployer(ctx, contractID, network)
	if err != nil {
		return "", err
	}

	if err := s.repo.Upsert(ctx, contractID, network, deployer); err != nil {
		return "", err
	}
	return deployer, nil
}

func (s *ContractAuthorityService) lookupDeployer(
	ctx context.Context,
	contractID, network string,
) (string, error) {
	expertNetwork := stellarExpertNetwork(network)
	url := fmt.Sprintf(
		"https://api.stellar.expert/explorer/%s/contract/%s",
		expertNetwork,
		contractID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build contract authority request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrContractAuthorityUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%w: contract not indexed", ErrContractAuthorityUnavailable)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf(
			"%w: stellar expert returned %d: %s",
			ErrContractAuthorityUnavailable,
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	var payload struct {
		Creator string `json:"creator"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("%w: decode response: %v", ErrContractAuthorityUnavailable, err)
	}

	deployer := strings.TrimSpace(payload.Creator)
	if deployer == "" || !strings.HasPrefix(deployer, "G") {
		return "", fmt.Errorf("%w: deployer wallet missing", ErrContractAuthorityUnavailable)
	}
	return deployer, nil
}

func stellarExpertNetwork(network string) string {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "mainnet":
		return "public"
	case "futurenet":
		return "future"
	default:
		return "testnet"
	}
}
