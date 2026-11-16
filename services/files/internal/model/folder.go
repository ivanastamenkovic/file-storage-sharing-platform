package model

import "time"

type Folder struct {
	ID        string    `bson:"_id" json:"id"`
	OwnerID   string    `bson:"owner_id" json:"ownerId"`
	ParentID  string    `bson:"parent_id" json:"parentId"`
	Name      string    `bson:"name" json:"name"`
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"`
}
