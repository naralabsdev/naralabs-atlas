package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	authservice "github.com/naralabs/naralabs-atlas/internal/module/auth/service"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/repository"
)

var (
	ErrInvalidProjectName = errors.New("invalid project name")
	ErrProjectNotFound    = errors.New("project not found")
)

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type ProjectRepository interface {
	Insert(ctx context.Context, userID, name, slug, description, status, publishTokenID, publishToken string) (model.SchemaProject, error)
	ListByUser(ctx context.Context, userID string) ([]model.SchemaProject, error)
	GetByIDForUser(ctx context.Context, projectID, userID string) (model.SchemaProject, error)
	Update(ctx context.Context, projectID, userID, name, slug, description string) (model.SchemaProject, error)
	MarkPublishedByPublishTokenID(ctx context.Context, publishTokenID, contractID, network string) error
	SyncPublishedStatus(ctx context.Context, projectID, userID string) (model.SchemaProject, error)
	SlugExists(ctx context.Context, userID, slug string) (bool, error)
}

type PublishTokenCreator interface {
	Create(ctx context.Context, authorization, label string) (tokenID string, rawToken string, err error)
}

type AuthUserParser interface {
	ParseUserID(raw string) (string, error)
}

type tokenCreatorAdapter struct {
	svc *authservice.PublishTokenService
}

func NewTokenCreatorAdapter(svc *authservice.PublishTokenService) PublishTokenCreator {
	return &tokenCreatorAdapter{svc: svc}
}

func (a *tokenCreatorAdapter) Create(ctx context.Context, authorization, label string) (string, string, error) {
	result, err := a.svc.Create(ctx, authorization, label)
	if err != nil {
		return "", "", err
	}
	return result.ID, result.Token, nil
}

type ProjectService struct {
	repo   ProjectRepository
	tokens PublishTokenCreator
	auth   AuthUserParser
}

func NewProjectService(repo ProjectRepository, tokens PublishTokenCreator, auth AuthUserParser) *ProjectService {
	return &ProjectService{repo: repo, tokens: tokens, auth: auth}
}

func (s *ProjectService) Create(
	ctx context.Context,
	authorization, name string,
) (model.SchemaProjectDetail, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.SchemaProjectDetail{}, ErrUnauthorized
	}

	name = repository.NormalizeProjectName(name)
	if len(name) < 2 || len(name) > 80 {
		return model.SchemaProjectDetail{}, ErrInvalidProjectName
	}

	slug, err := s.uniqueSlug(ctx, userID, slugify(name))
	if err != nil {
		return model.SchemaProjectDetail{}, err
	}

	tokenID, rawToken, err := s.tokens.Create(ctx, authorization, name)
	if err != nil {
		return model.SchemaProjectDetail{}, err
	}

	project, err := s.repo.Insert(
		ctx,
		userID,
		name,
		slug,
		"",
		"draft",
		tokenID,
		rawToken,
	)
	if err != nil {
		return model.SchemaProjectDetail{}, err
	}

	return project.Detail(), nil
}

func (s *ProjectService) List(ctx context.Context, authorization string) (model.SchemaProjectListResponse, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.SchemaProjectListResponse{}, ErrUnauthorized
	}

	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return model.SchemaProjectListResponse{}, err
	}

	out := model.SchemaProjectListResponse{
		Total: len(items),
		Items: make([]model.SchemaProjectPublic, 0, len(items)),
	}
	for _, item := range items {
		item, err = s.syncPublishedStatus(ctx, item)
		if err != nil {
			return model.SchemaProjectListResponse{}, err
		}
		out.Items = append(out.Items, item.Public())
	}
	return out, nil
}

func (s *ProjectService) MarkPublishedByPublishTokenID(
	ctx context.Context,
	publishTokenID, contractID, network string,
) error {
	publishTokenID = strings.TrimSpace(publishTokenID)
	if publishTokenID == "" {
		return nil
	}
	return s.repo.MarkPublishedByPublishTokenID(ctx, publishTokenID, contractID, network)
}

func (s *ProjectService) Get(
	ctx context.Context,
	authorization, projectID string,
) (model.SchemaProjectDetail, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.SchemaProjectDetail{}, ErrUnauthorized
	}

	project, err := s.repo.GetByIDForUser(ctx, projectID, userID)
	if errors.Is(err, repository.ErrSchemaProjectNotFound) {
		return model.SchemaProjectDetail{}, ErrProjectNotFound
	}
	if err != nil {
		return model.SchemaProjectDetail{}, err
	}

	project, err = s.syncPublishedStatus(ctx, project)
	if err != nil {
		return model.SchemaProjectDetail{}, err
	}

	return project.Detail(), nil
}

func (s *ProjectService) Update(
	ctx context.Context,
	authorization, projectID, name, description string,
) (model.SchemaProjectDetail, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.SchemaProjectDetail{}, ErrUnauthorized
	}

	project, err := s.repo.GetByIDForUser(ctx, projectID, userID)
	if errors.Is(err, repository.ErrSchemaProjectNotFound) {
		return model.SchemaProjectDetail{}, ErrProjectNotFound
	}
	if err != nil {
		return model.SchemaProjectDetail{}, err
	}

	nextName := project.Name
	if trimmed := repository.NormalizeProjectName(name); trimmed != "" {
		if len(trimmed) < 2 || len(trimmed) > 80 {
			return model.SchemaProjectDetail{}, ErrInvalidProjectName
		}
		nextName = trimmed
	}

	nextSlug := project.Slug
	if nextName != project.Name {
		nextSlug, err = s.uniqueSlug(ctx, userID, slugify(nextName))
		if err != nil {
			return model.SchemaProjectDetail{}, err
		}
	}

	updated, err := s.repo.Update(ctx, projectID, userID, nextName, nextSlug, strings.TrimSpace(description))
	if errors.Is(err, repository.ErrSchemaProjectNotFound) {
		return model.SchemaProjectDetail{}, ErrProjectNotFound
	}
	if err != nil {
		return model.SchemaProjectDetail{}, err
	}

	return updated.Detail(), nil
}

func (s *ProjectService) syncPublishedStatus(
	ctx context.Context,
	project model.SchemaProject,
) (model.SchemaProject, error) {
	if project.Status != "draft" {
		return project, nil
	}

	updated, err := s.repo.SyncPublishedStatus(ctx, project.ID, project.UserID)
	if errors.Is(err, repository.ErrSchemaProjectNotFound) {
		return project, nil
	}
	if err != nil {
		return model.SchemaProject{}, err
	}
	return updated, nil
}

func (s *ProjectService) uniqueSlug(ctx context.Context, userID, base string) (string, error) {
	candidate := base
	for i := 0; i < 20; i++ {
		exists, err := s.repo.SlugExists(ctx, userID, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i+2)
	}
	return "", ErrInvalidProjectName
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugSanitizer.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "schema"
	}
	return s
}
