package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID        int64     `bson:"_id" json:"id"`
	Username  string    `bson:"username" json:"username"`
	Password  string    `bson:"password" json:"-"`
	Email     string    `bson:"email" json:"email"`
	Nickname  string    `bson:"nickname" json:"nickname"`
	Avatar    string    `bson:"avatar" json:"avatar"`
	Status    int       `bson:"status" json:"status"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
} 