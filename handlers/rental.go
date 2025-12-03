package handlers

import (
	"kbt-cuy/esp32"
	"kbt-cuy/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RentalHandler struct {
	DB *gorm.DB
}

// ShowRentalStations displays available stations
func (h *RentalHandler) ShowRentalStations(c *gin.Context) {
	var stations []models.PowerbankStation
	// Only show stations with powerbanks
	h.DB.Where("powerbank_left > ?", 0).Find(&stations)

	c.HTML(http.StatusOK, "rental.html", gin.H{
		"Stations":   stations,
		"IsLoggedIn": true,
	})
}

// RentPowerbank handles the logic
func (h *RentalHandler) RentPowerbank(c *gin.Context) {
	stationID, _ := strconv.Atoi(c.PostForm("station_id"))
	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	// Transaction to ensure atomicity
	tx := h.DB.Begin()

	var station models.PowerbankStation
	if err := tx.First(&station, stationID).Error; err != nil {
		tx.Rollback()
		c.String(http.StatusBadRequest, "Station not found")
		return
	}

	if station.PowerbankLeft <= 0 {
		tx.Rollback()
		c.String(http.StatusBadRequest, "No powerbanks available")
		return
	}

	// Find an available powerbank at this station
	var pb models.Powerbank
	if err := tx.Where("current_station_id = ? AND status = ?", stationID, "Available").First(&pb).Error; err != nil {
		tx.Rollback()
		c.String(http.StatusInternalServerError, "Data inconsistency error")
		return
	}

	// Create Transaction
	transaction := models.Transaction{
		UserID:                   userID,
		PowerbankID:              pb.ID,
		PowerbankStationOriginID: station.ID,
		Status:                   "Ongoing",
	}

	// Update Data
	pb.Status = "Rented"
	pb.CurrentStationID = nil // It is now with the user
	station.PowerbankLeft--

	if err := tx.Save(&pb).Error; err != nil {
		tx.Rollback()
		return
	}
	if err := tx.Save(&station).Error; err != nil {
		tx.Rollback()
		return
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()

	// Hardware Trigger
	esp32.TriggerLock(station.IPAddress, "open")

	c.Redirect(http.StatusFound, "/account")
}

// ShowReturnStations displays stations with empty slots
func (h *RentalHandler) ShowReturnStations(c *gin.Context) {
	var stations []models.PowerbankStation
	// Only show stations with space
	h.DB.Where("powerbank_left < capacity").Find(&stations)

	// Check if user actually has an ongoing rental
	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)
	var activeTx models.Transaction
	hasActive := h.DB.Where("user_id = ? AND status = ?", userID, "Ongoing").First(&activeTx).RowsAffected > 0

	c.HTML(http.StatusOK, "return.html", gin.H{
		"Stations":        stations,
		"HasActiveRental": hasActive,
		"ActiveTxID":      activeTx.ID,
		"IsLoggedIn":      true,
	})
}

// ReturnPowerbank handles the return logic
func (h *RentalHandler) ReturnPowerbank(c *gin.Context) {
	stationID, _ := strconv.Atoi(c.PostForm("station_id"))
	txID, _ := strconv.Atoi(c.PostForm("transaction_id"))

	txDB := h.DB.Begin()

	var transaction models.Transaction
	if err := txDB.Preload("Powerbank").First(&transaction, txID).Error; err != nil {
		txDB.Rollback()
		c.String(http.StatusBadRequest, "Transaction not found")
		return
	}

	var station models.PowerbankStation
	txDB.First(&station, stationID)

	// Update Transaction
	now := time.Now()
	rtnStationID := uint(stationID)
	transaction.PowerbankStationReturnID = &rtnStationID
	transaction.DateReturned = &now
	transaction.Status = "Returned"

	// Update Powerbank
	transaction.Powerbank.Status = "Available"
	stID := uint(station.ID)
	transaction.Powerbank.CurrentStationID = &stID

	// Update Station
	station.PowerbankLeft++

	txDB.Save(&transaction)
	txDB.Save(&transaction.Powerbank)
	txDB.Save(&station)
	txDB.Commit()

	// Hardware Trigger
	esp32.TriggerLock(station.IPAddress, "open")

	c.Redirect(http.StatusFound, "/account")
}
