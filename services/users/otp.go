package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrOTPInvalid        = errors.New("invalid or already used code")
	ErrOTPExpired        = errors.New("code expired")
	ErrOTPTooManyAttempt = errors.New("too many attempts")
)

// loginChallenge is created after a correct email + password and is
// completed by proving possession of the OTP sent by email.
type loginChallenge struct {
	ID        string     `bson:"_id"`
	UserID    string     `bson:"userId"`
	OTPHash   string     `bson:"otpHash"`
	ExpiresAt time.Time  `bson:"expiresAt"`
	Attempts  int        `bson:"attempts"`
	Used      bool       `bson:"used"`
	CreatedAt time.Time  `bson:"createdAt"`
	UsedAt    *time.Time `bson:"usedAt,omitempty"`
}

type challengeStore struct {
	challenges *mongo.Collection
}

func newChallengeStore(ctx context.Context, db *mongo.Database) (*challengeStore, error) {
	c := db.Collection("login_challenges")
	// Mongo removes challenges an hour after they expire; until then an
	// expired code still gets a clear "expired" answer instead of "invalid".
	_, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(3600).SetName("expires_ttl"),
	})
	if err != nil {
		return nil, err
	}
	return &challengeStore{challenges: c}, nil
}

func newOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashOTP(challengeID, otp string) string {
	sum := sha256.Sum256([]byte(challengeID + ":" + otp))
	return hex.EncodeToString(sum[:])
}

func (s *challengeStore) Create(ctx context.Context, c *loginChallenge) error {
	_, err := s.challenges.InsertOne(ctx, c)
	return err
}

// Consume checks the code and marks the challenge as used. It returns the
// user id on success, and the number of attempts left on a wrong code.
func (s *challengeStore) Consume(ctx context.Context, id, otp string, now time.Time, maxAttempts int) (string, int, error) {
	var c loginChallenge
	// Count the attempt atomically before comparing, so parallel guesses
	// cannot get around the attempt limit.
	err := s.challenges.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "used", Value: false}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "attempts", Value: 1}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", 0, ErrOTPInvalid
	}
	if err != nil {
		return "", 0, err
	}

	if !now.Before(c.ExpiresAt) {
		return "", 0, ErrOTPExpired
	}
	if c.Attempts > maxAttempts {
		_, _ = s.challenges.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "used", Value: true}}}})
		return "", 0, ErrOTPTooManyAttempt
	}
	if subtle.ConstantTimeCompare([]byte(hashOTP(id, otp)), []byte(c.OTPHash)) != 1 {
		return "", maxAttempts - c.Attempts, ErrOTPInvalid
	}

	res, err := s.challenges.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "used", Value: false}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "used", Value: true}, {Key: "usedAt", Value: now}}}},
	)
	if err != nil {
		return "", 0, err
	}
	if res.ModifiedCount == 0 {
		// Another request used this code first.
		return "", 0, ErrOTPInvalid
	}
	return c.UserID, 0, nil
}
