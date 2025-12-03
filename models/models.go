package models

import (
	"time"

	"gorm.io/gorm"
)

// User stores authentication and profile details
type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;not null"`
	Email        string `gorm:"uniqueIndex;not null"`
	Password     string `gorm:"not null"` // Hashed
	Transactions []Transaction
}

// PowerbankStation stores location and capacity
type PowerbankStation struct {
	gorm.Model
	Name          string
	Latitude      float64
	Longitude     float64
	Capacity      int
	PowerbankLeft int
	IPAddress     string // For ESP32 communication
	// FIX: Explicitly specify that the Foreign Key in the Powerbank struct is 'CurrentStationID'
	Powerbanks []Powerbank `gorm:"foreignKey:CurrentStationID"`
}

// Powerbank represents the physical unit
type Powerbank struct {
	gorm.Model
	PowerbankCode    string `gorm:"uniqueIndex"`
	Capacity         int    // e.g., 10000mAh
	Status           string // "Available", "Rented"
	CurrentStationID *uint
	CurrentStation   *PowerbankStation `gorm:"foreignKey:CurrentStationID"`
}

// Transaction records the rental history
type Transaction struct {
	gorm.Model
	UserID                   uint
	User                     User
	PowerbankID              uint
	Powerbank                Powerbank
	PowerbankStationOriginID uint
	PowerbankStationOrigin   PowerbankStation `gorm:"foreignKey:PowerbankStationOriginID"`
	PowerbankStationReturnID *uint
	PowerbankStationReturn   *PowerbankStation `gorm:"foreignKey:PowerbankStationReturnID"`
	Status                   string            // "Ongoing", "Returned"
	DateReturned             *time.Time
}
