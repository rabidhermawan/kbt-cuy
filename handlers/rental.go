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
	h.DB.Where("powerbank_left > ?", 0).Find(&stations)

	c.HTML(http.StatusOK, "rental.html", gin.H{
		"Stations":   stations,
		"IsLoggedIn": true,
	})
}

// ShowPaymentPage renders the payment confirmation screen
func (h *RentalHandler) ShowPaymentPage(c *gin.Context) {
	stationID := c.Param("id")
	var station models.PowerbankStation
	if err := h.DB.First(&station, stationID).Error; err != nil {
		c.String(http.StatusNotFound, "Station not found")
		return
	}

	c.HTML(http.StatusOK, "payment.html", gin.H{
		"Station":    station,
		"IsLoggedIn": true,
	})
}

// RentPowerbank processes payment, creates transaction, and auto-triggers door
func (h *RentalHandler) RentPowerbank(c *gin.Context) {
	stationID, _ := strconv.Atoi(c.PostForm("station_id"))
	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	// Start Transaction
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

	// Find available powerbank
	var pb models.Powerbank
	if err := tx.Where("current_station_id = ? AND status = ?", stationID, "Available").First(&pb).Error; err != nil {
		tx.Rollback()
		c.String(http.StatusInternalServerError, "Data inconsistency error")
		return
	}

	// Create Transaction Record
	transaction := models.Transaction{
		UserID:                   userID,
		PowerbankID:              pb.ID,
		PowerbankStationOriginID: station.ID,
		Status:                   "Ongoing",
	}

	// Update Statuses
	pb.Status = "Rented"
	pb.CurrentStationID = nil
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

	// AUTOMATIC TRIGGER: Open the door immediately upon successful processing
	go esp32.TriggerLock(station.IPAddress, "open")

	// Render Success Page
	c.HTML(http.StatusOK, "rental_success.html", gin.H{
		"TransactionID": transaction.ID,
		"StationName":   station.Name,
		"IsLoggedIn":    true,
	})
}

// RetryOpenDoor allows manual triggering from the success page
func (h *RentalHandler) RetryOpenDoor(c *gin.Context) {
	txID, _ := strconv.Atoi(c.PostForm("transaction_id"))
	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	var transaction models.Transaction
	// Validate that this transaction belongs to the logged-in user and is recent/ongoing
	if err := h.DB.Preload("PowerbankStationOrigin").
		Where("id = ? AND user_id = ? AND status = ?", txID, userID, "Ongoing").
		First(&transaction).Error; err != nil {
		c.String(http.StatusBadRequest, "Invalid request or transaction not found")
		return
	}

	// Trigger the lock again
	err := esp32.RetryOpen(transaction.PowerbankStationOrigin.IPAddress)

	msg := "Door open signal sent!"
	if err != nil {
		msg = "Failed to connect to station."
	}

	// Re-render success page with message
	c.HTML(http.StatusOK, "rental_success.html", gin.H{
		"TransactionID": transaction.ID,
		"StationName":   transaction.PowerbankStationOrigin.Name,
		"Message":       msg,
		"IsLoggedIn":    true,
	})
}

// ShowReturnStations displays stations with empty slots
func (h *RentalHandler) ShowReturnStations(c *gin.Context) {
	var stations []models.PowerbankStation
	h.DB.Where("powerbank_left < capacity").Find(&stations)

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

	now := time.Now()
	rtnStationID := uint(stationID)
	transaction.PowerbankStationReturnID = &rtnStationID
	transaction.DateReturned = &now
	transaction.Status = "Returned"

	transaction.Powerbank.Status = "Available"
	stID := uint(station.ID)
	transaction.Powerbank.CurrentStationID = &stID

	station.PowerbankLeft++

	txDB.Save(&transaction)
	txDB.Save(&transaction.Powerbank)
	txDB.Save(&station)
	txDB.Commit()

	esp32.TriggerLock(station.IPAddress, "open")

	c.Redirect(http.StatusFound, "/account")
}
