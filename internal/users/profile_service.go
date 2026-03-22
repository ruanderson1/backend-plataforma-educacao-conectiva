package users

import (
	"context"
)

type ProfileService struct {
	repo *ProfileRepository
}

func NewProfileService(repo *ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) SaveProfile(ctx context.Context, profile Profile) error {
	return s.repo.Upsert(ctx, profile)
}

func (s *ProfileService) GetProfile(ctx context.Context, publicID string) (Profile, error) {
	return s.repo.GetByPublicID(ctx, publicID)
}
