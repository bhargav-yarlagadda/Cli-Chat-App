package db

import (
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB_Conn *gorm.DB
func ConnectToDB()error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	dbUrl := os.Getenv("DB_URL")
	db,err := gorm.Open(postgres.Open(dbUrl),&gorm.Config{})
	// create table if not exists or update it if any columns changes
    if err := db.AutoMigrate(&User{}, &Connection{}, &Message{}); err != nil {
		return err
	}
	DB_Conn = db 
	return nil
}