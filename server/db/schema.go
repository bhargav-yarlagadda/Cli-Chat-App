package db

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`               // Primary key column, auto-increment
	Username  string    `gorm:"uniqueIndex;not null" json:"username"` // Unique and required column
	Password  string    `gorm:"not null" json:"password"`           // Required column, will store hashed passwords
	PublicKey string    `gorm:"not null" json:"public_key"`         // Required column for storing public key
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`   // Automatically set when a new row is created
}