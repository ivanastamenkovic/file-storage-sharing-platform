package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/model"
	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/repository"
)

// FolderStore je ono što servisu treba od baze.
// Interfejs olakšava testiranje, jer se u testu može podmetnuti lažna implementacija.
type FolderStore interface {
	Create(ctx context.Context, f *model.Folder) error
	FindByID(ctx context.Context, id string) (*model.Folder, error)
}

type FolderService struct {
	store FolderStore
}

func NewFolderService(store FolderStore) *FolderService {
	return &FolderService{store: store}
}

func (s *FolderService) CreateFolder(ctx context.Context, ownerID, parentID, name string) (*model.Folder, error) {
	name, err := model.ValidateFolderName(name)
	if err != nil {
		return nil, err
	}

	if parentID != "" {
		parent, err := s.store.FindByID(ctx, parentID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, model.ErrParentNotFound
		}
		if err != nil {
			return nil, err
		}

		if parent.OwnerID != ownerID {
			return nil, model.ErrParentNotFound
		}
	}

	now := time.Now().UTC()
	folder := &model.Folder{
		ID:        uuid.NewString(),
		OwnerID:   ownerID,
		ParentID:  parentID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}
