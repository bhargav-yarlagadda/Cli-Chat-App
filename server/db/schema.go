package db

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`               // Primary key column, auto-increment
	Username  string    `gorm:"uniqueIndex;not null" json:"username"` // Unique and required column
	Password  string    `gorm:"not null" json:"password"`           // Required column, will store hashed passwords
	PublicKey string    `gorm:"not null" json:"public_key"`         // Required column for storing public key
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`   // Automatically set when a new row is created
}

type Connection struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	SenderID   uint   `gorm:"not null;uniqueIndex:idx_sender_receiver" json:"sender_id"`
	ReceiverID uint   `gorm:"not null;uniqueIndex:idx_sender_receiver" json:"receiver_id"`
	Status     string `gorm:"type:varchar(20);not null;default:'pending'" json:"status,omitempty"`

	Sender   User `gorm:"foreignKey:SenderID" json:"-"`
	Receiver User `gorm:"foreignKey:ReceiverID" json:"-"`
}


type Message struct {
    ID         uint      `gorm:"primaryKey" json:"id"`
    SenderID   uint      `gorm:"not null" json:"sender_id"`
    ReceiverID uint      `gorm:"not null" json:"receiver_id"`
    Content    string    `gorm:"not null" json:"content"` // encrypted text
    Delivered  bool      `gorm:"default:false" json:"delivered"`
    CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

