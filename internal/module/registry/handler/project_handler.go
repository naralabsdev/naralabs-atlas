package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	authservice "github.com/naralabs/naralabs-atlas/internal/module/auth/service"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/service"
)

type ProjectVerifyService interface {
	ListPublications(ctx context.Context, authorization, projectID string) (model.ProjectPublicationsResponse, error)
	CreateChallenge(
		ctx context.Context,
		authorization, projectID string,
		input model.VerifyChallengeInput,
	) (model.VerifyChallengeResponse, error)
	VerifyContract(
		ctx context.Context,
		authorization, projectID string,
		input model.VerifyContractInput,
	) (model.VerifyContractResponse, error)
}

type ProjectHandler struct {
	svc    *service.ProjectService
	verify ProjectVerifyService
}

func NewProjectHandler(svc *service.ProjectService, verify ProjectVerifyService) *ProjectHandler {
	return &ProjectHandler{svc: svc, verify: verify}
}

func (h *ProjectHandler) HandleCreate(ctx context.Context, input *CreateProjectInput) (*CreateProjectOutput, error) {
	project, err := h.svc.Create(ctx, input.Authorization, input.Body.Name)
	if err != nil {
		return nil, mapProjectError(err)
	}

	out := &CreateProjectOutput{}
	out.Body = project
	return out, nil
}

func (h *ProjectHandler) HandleList(ctx context.Context, input *ListProjectsInput) (*ListProjectsOutput, error) {
	result, err := h.svc.List(ctx, input.Authorization)
	if err != nil {
		return nil, mapProjectError(err)
	}
	return &ListProjectsOutput{Body: result}, nil
}

func (h *ProjectHandler) HandleGet(ctx context.Context, input *GetProjectInput) (*GetProjectOutput, error) {
	project, err := h.svc.Get(ctx, input.Authorization, input.ID)
	if err != nil {
		return nil, mapProjectError(err)
	}
	return &GetProjectOutput{Body: project}, nil
}

func (h *ProjectHandler) HandleUpdate(ctx context.Context, input *UpdateProjectInput) (*UpdateProjectOutput, error) {
	project, err := h.svc.Update(
		ctx,
		input.Authorization,
		input.ID,
		input.Body.Name,
		input.Body.Description,
	)
	if err != nil {
		return nil, mapProjectError(err)
	}
	return &UpdateProjectOutput{Body: project}, nil
}

func (h *ProjectHandler) HandleListPublications(
	ctx context.Context,
	input *ListPublicationsInput,
) (*ListPublicationsOutput, error) {
	if h.verify == nil {
		return nil, huma.Error500InternalServerError("publications unavailable")
	}

	result, err := h.verify.ListPublications(ctx, input.Authorization, input.ID)
	if err != nil {
		return nil, mapVerifyError(err)
	}
	return &ListPublicationsOutput{Body: result}, nil
}

func (h *ProjectHandler) HandleVerifyChallenge(
	ctx context.Context,
	input *VerifyChallengeInput,
) (*VerifyChallengeOutput, error) {
	if h.verify == nil {
		return nil, huma.Error500InternalServerError("verify unavailable")
	}

	result, err := h.verify.CreateChallenge(ctx, input.Authorization, input.ID, model.VerifyChallengeInput{
		ContractID: input.Body.ContractID,
		Network:    input.Body.Network,
	})
	if err != nil {
		return nil, mapVerifyError(err)
	}
	return &VerifyChallengeOutput{Body: result}, nil
}

func (h *ProjectHandler) HandleVerifyContract(
	ctx context.Context,
	input *VerifyContractInput,
) (*VerifyContractOutput, error) {
	if h.verify == nil {
		return nil, huma.Error500InternalServerError("verify unavailable")
	}

	result, err := h.verify.VerifyContract(ctx, input.Authorization, input.ID, model.VerifyContractInput{
		ContractID: input.Body.ContractID,
		Network:    input.Body.Network,
		Wallet:     input.Body.Wallet,
		Signature:  input.Body.Signature,
		Nonce:      input.Body.Nonce,
	})
	if err != nil {
		return nil, mapVerifyError(err)
	}
	return &VerifyContractOutput{Body: result}, nil
}

func mapProjectError(err error) error {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		return schemaError(http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	case errors.Is(err, service.ErrInvalidProjectName):
		return schemaError(http.StatusBadRequest, "INVALID_PROJECT_NAME", "Schema name must be 2–80 characters")
	case errors.Is(err, service.ErrProjectNotFound):
		return huma.Error404NotFound("schema not found")
	default:
		if errors.Is(err, authservice.ErrPublishTokenLimit) {
			return schemaError(http.StatusConflict, "PUBLISH_TOKEN_LIMIT", "Maximum number of publish tokens reached")
		}
		if msg := err.Error(); strings.Contains(msg, "invalid") {
			return huma.Error400BadRequest(msg)
		}
		return huma.Error500InternalServerError("schema project request failed", err)
	}
}

func mapVerifyError(err error) error {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		return schemaError(http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	case errors.Is(err, service.ErrProjectNotFound):
		return huma.Error404NotFound("schema not found")
	case errors.Is(err, service.ErrInvalidContractID):
		return schemaError(http.StatusBadRequest, "INVALID_CONTRACT_ID", "contractId must be a valid Soroban contract address")
	case errors.Is(err, service.ErrInvalidNetwork):
		return schemaError(http.StatusBadRequest, "INVALID_NETWORK", "network must be testnet, mainnet, or futurenet")
	case errors.Is(err, service.ErrInvalidSignature):
		return schemaError(http.StatusBadRequest, "INVALID_SIGNATURE", "Wallet signature verification failed")
	case errors.Is(err, service.ErrChallengeNotFound):
		return schemaError(http.StatusBadRequest, "CHALLENGE_NOT_FOUND", "Verify challenge not found or expired")
	case errors.Is(err, service.ErrChallengeExpired):
		return schemaError(http.StatusBadRequest, "CHALLENGE_EXPIRED", "Verify challenge expired")
	case errors.Is(err, service.ErrChallengeAlreadyUsed):
		return schemaError(http.StatusBadRequest, "CHALLENGE_USED", "Verify challenge already used")
	case errors.Is(err, service.ErrContractAuthorityDenied):
		return schemaError(http.StatusForbidden, "CONTRACT_AUTHORITY_DENIED", "Wallet is not authorized for this contract")
	case errors.Is(err, service.ErrContractAuthorityUnavailable):
		return schemaError(http.StatusForbidden, "CONTRACT_AUTHORITY_UNAVAILABLE", "Unable to resolve contract deployer authority")
	case errors.Is(err, service.ErrNothingToVerify):
		return schemaError(http.StatusBadRequest, "NOTHING_TO_VERIFY", "No community schemas available to verify for this contract")
	default:
		if msg := err.Error(); strings.Contains(msg, "invalid") {
			return huma.Error400BadRequest(msg)
		}
		return huma.Error500InternalServerError("verify request failed", err)
	}
}

type CreateProjectInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	Body          struct {
		Name string `json:"name" doc:"Schema name" minLength:"2" maxLength:"80"`
	}
}

type CreateProjectOutput struct {
	Body model.SchemaProjectDetail
}

type ListProjectsInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
}

type ListProjectsOutput struct {
	Body model.SchemaProjectListResponse
}

type GetProjectInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	ID            string `path:"id" doc:"Schema project ID"`
}

type GetProjectOutput struct {
	Body model.SchemaProjectDetail
}

type UpdateProjectInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	ID            string `path:"id" doc:"Schema project ID"`
	Body          struct {
		Name        string `json:"name,omitempty" doc:"Schema name" maxLength:"80"`
		Description string `json:"description,omitempty" doc:"Optional description" maxLength:"500"`
	}
}

type UpdateProjectOutput struct {
	Body model.SchemaProjectDetail
}

type ListPublicationsInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	ID            string `path:"id" doc:"Schema project ID"`
}

type ListPublicationsOutput struct {
	Body model.ProjectPublicationsResponse
}

type VerifyChallengeInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	ID            string `path:"id" doc:"Schema project ID"`
	Body          struct {
		ContractID string `json:"contractId" doc:"Soroban contract address"`
		Network    string `json:"network,omitempty" doc:"Stellar network" enum:"testnet,mainnet,futurenet"`
	}
}

type VerifyChallengeOutput struct {
	Body model.VerifyChallengeResponse
}

type VerifyContractInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	ID            string `path:"id" doc:"Schema project ID"`
	Body          struct {
		ContractID string `json:"contractId" doc:"Soroban contract address"`
		Network    string `json:"network,omitempty" doc:"Stellar network" enum:"testnet,mainnet,futurenet"`
		Wallet     string `json:"wallet" doc:"Connected wallet public key"`
		Signature  string `json:"signature" doc:"Base64 Ed25519 signature over the challenge message"`
		Nonce      string `json:"nonce" doc:"Challenge nonce"`
	}
}

type VerifyContractOutput struct {
	Body model.VerifyContractResponse
}
