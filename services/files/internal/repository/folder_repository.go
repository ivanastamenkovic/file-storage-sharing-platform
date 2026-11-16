package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/model"
)

const foldersCollection = "folders"

var ErrNotFound = errors.New("not found")

type FolderRepository struct {
	coll *mongo.Collection
}

func NewFolderRepository(ctx context.Context, db *mongo.Database) (*FolderRepository, error) {
	coll := db.Collection(foldersCollection)

	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "owner_id", Value: 1},
			{Key: "parent_id", Value: 1},
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	return &FolderRepository{coll: coll}, nil
}

func (r *FolderRepository) Create(ctx context.Context, f *model.Folder) error {
	_, err := r.coll.InsertOne(ctx, f)
	if mongo.IsDuplicateKeyError(err) {
		return model.ErrFolderExists
	}
	return err
}

func (r *FolderRepository) FindByID(ctx context.Context, id string) (*model.Folder, error) {
	var f model.Folder
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&f)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
