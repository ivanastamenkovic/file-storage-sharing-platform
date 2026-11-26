package main

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	StatusPending = "PENDING"
	StatusActive  = "ACTIVE"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidToken = errors.New("invalid confirmation token")
	ErrTokenExpired = errors.New("confirmation token expired")
)

type confirmation struct {
	TokenHash string    `bson:"tokenHash"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

type User struct {
	ID                 string        `bson:"_id"`
	FirstName          string        `bson:"firstName"`
	LastName           string        `bson:"lastName"`
	Email              string        `bson:"email"`
	PasswordHash       string        `bson:"passwordHash"`
	Status             string        `bson:"status"`
	StorageQuotaBytes  int64         `bson:"storageQuotaBytes"`
	StorageUsedBytes   int64         `bson:"storageUsedBytes"`
	CreatedAt          time.Time     `bson:"createdAt"`
	PasswordChangedAt  time.Time     `bson:"passwordChangedAt"`
	MustChangePassword bool          `bson:"mustChangePassword"`
	ActivatedAt        *time.Time    `bson:"activatedAt,omitempty"`
	Confirmation       *confirmation `bson:"confirmation,omitempty"`
}

// PasswordSetAt returns when the current password was set. Accounts created
// before this field existed fall back to their creation time.
func (u *User) PasswordSetAt() time.Time {
	if u.PasswordChangedAt.IsZero() {
		return u.CreatedAt
	}
	return u.PasswordChangedAt
}

type userStore struct {
	users *mongo.Collection
}

func connectMongo(uri string) *mongo.Client {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("invalid mongo configuration: %v", err)
	}

	const maxAttempts = 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = client.Ping(ctx, nil)
		cancel()
		if err == nil {
			log.Printf("connected to mongo (attempt %d)", attempt)
			return client
		}
		log.Printf("mongo not ready (attempt %d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(3 * time.Second)
	}
	log.Fatal("could not connect to mongo")
	return nil
}

func newUserStore(ctx context.Context, db *mongo.Database) (*userStore, error) {
	users := db.Collection("users")
	_, err := users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("email_unique"),
		},
		{
			Keys:    bson.D{{Key: "confirmation.tokenHash", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("confirmation_token"),
		},
	})
	if err != nil {
		return nil, err
	}
	return &userStore{users: users}, nil
}

// Create inserts a new user. The unique index on email makes the
// "already registered" check atomic, even under concurrent requests.
func (s *userStore) Create(ctx context.Context, u *User) error {
	_, err := s.users.InsertOne(ctx, u)
	if mongo.IsDuplicateKeyError(err) {
		return ErrEmailTaken
	}
	return err
}

func (s *userStore) findOne(ctx context.Context, filter bson.D) (*User, error) {
	var u User
	err := s.users.FindOne(ctx, filter).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *userStore) FindByEmail(ctx context.Context, email string) (*User, error) {
	return s.findOne(ctx, bson.D{{Key: "email", Value: email}})
}

func (s *userStore) FindByID(ctx context.Context, id string) (*User, error) {
	return s.findOne(ctx, bson.D{{Key: "_id", Value: id}})
}

func (s *userStore) Delete(ctx context.Context, id string) error {
	_, err := s.users.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	return err
}

// SetPassword stores a new password hash, records when it was changed and
// clears the "must change password" flag.
func (s *userStore) SetPassword(ctx context.Context, id, passwordHash string, now time.Time) error {
	res, err := s.users.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "passwordHash", Value: passwordHash},
			{Key: "passwordChangedAt", Value: now},
			{Key: "mustChangePassword", Value: false},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Activate moves a pending account to ACTIVE if the token matches and has not expired.
func (s *userStore) Activate(ctx context.Context, tokenHash string, now time.Time) error {
	var u User
	err := s.users.FindOne(ctx, bson.D{
		{Key: "confirmation.tokenHash", Value: tokenHash},
		{Key: "status", Value: StatusPending},
	}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrInvalidToken
	}
	if err != nil {
		return err
	}
	if !now.Before(u.Confirmation.ExpiresAt) {
		return ErrTokenExpired
	}

	res, err := s.users.UpdateOne(ctx,
		bson.D{
			{Key: "_id", Value: u.ID},
			{Key: "status", Value: StatusPending},
			{Key: "confirmation.tokenHash", Value: tokenHash},
		},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "status", Value: StatusActive},
				{Key: "activatedAt", Value: now},
			}},
			{Key: "$unset", Value: bson.D{{Key: "confirmation", Value: ""}}},
		},
	)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		// Confirmed concurrently by another request.
		return ErrInvalidToken
	}
	return nil
}
