package main

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	TokenPasswordReset = "password_reset"
	TokenMagicLink     = "magic_link"
)

var (
	ErrAuthTokenInvalid = errors.New("invalid or already used token")
	ErrAuthTokenExpired = errors.New("token expired")
)

type authToken struct {
	TokenHash string     `bson:"_id"`
	UserID    string     `bson:"userId"`
	Type      string     `bson:"type"`
	ExpiresAt time.Time  `bson:"expiresAt"`
	CreatedAt time.Time  `bson:"createdAt"`
	UsedAt    *time.Time `bson:"usedAt,omitempty"`
}

type authTokenStore struct {
	tokens *mongo.Collection
}

func newAuthTokenStore(ctx context.Context, db *mongo.Database) (*authTokenStore, error) {
	c := db.Collection("auth_tokens")
	_, err := c.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(3600).SetName("expires_ttl"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "type", Value: 1}},
			Options: options.Index().SetName("user_type"),
		},
	})
	if err != nil {
		return nil, err
	}
	return &authTokenStore{tokens: c}, nil
}

func (s *authTokenStore) Create(ctx context.Context, t *authToken) error {
	_, err := s.tokens.InsertOne(ctx, t)
	return err
}

func (s *authTokenStore) InvalidateUnused(ctx context.Context, userID, tokenType string) error {
	_, err := s.tokens.DeleteMany(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "type", Value: tokenType},
		{Key: "usedAt", Value: bson.D{{Key: "$exists", Value: false}}},
	})
	return err
}

func (s *authTokenStore) Consume(ctx context.Context, tokenHash, tokenType string, now time.Time) (string, error) {
	var t authToken
	err := s.tokens.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: tokenHash},
			{Key: "type", Value: tokenType},
			{Key: "usedAt", Value: bson.D{{Key: "$exists", Value: false}}},
			{Key: "expiresAt", Value: bson.D{{Key: "$gt", Value: now}}},
		},
		bson.D{{Key: "$set", Value: bson.D{{Key: "usedAt", Value: now}}}},
	).Decode(&t)
	if err == nil {
		return t.UserID, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return "", err
	}

	err = s.tokens.FindOne(ctx, bson.D{
		{Key: "_id", Value: tokenHash},
		{Key: "type", Value: tokenType},
		{Key: "usedAt", Value: bson.D{{Key: "$exists", Value: false}}},
		{Key: "expiresAt", Value: bson.D{{Key: "$lte", Value: now}}},
	}).Err()
	if err == nil {
		return "", ErrAuthTokenExpired
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", ErrAuthTokenInvalid
	}
	return "", err
}

func (s *authTokenStore) CreatedSince(ctx context.Context, userID, tokenType string, since time.Time) (bool, error) {
	err := s.tokens.FindOne(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "type", Value: tokenType},
		{Key: "createdAt", Value: bson.D{{Key: "$gte", Value: since}}},
	}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
