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

// RentalSuccess page after successful payment and processing
func (h *RentalHandler) RentalSuccess(c *gin.Context) {
	txID := c.Param("id")
	var transaction models.Transaction
	if err := h.DB.Preload("PowerbankStationOrigin").First(&transaction, txID).Error; err != nil {
		c.String(http.StatusNotFound, "Transaction not found")
		return
	}

	c.HTML(http.StatusOK, "rental_success.html", gin.H{
		"TransactionID": transaction.ID,
		"StationName":   transaction.PowerbankStationOrigin.Name,
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

	// AUTOMATIC TRIGGER
	go esp32.TriggerLock(station.IPAddress, "open")

	// Render Success Page (Changed from Redirect)
	c.HTML(http.StatusOK, "return_success.html", gin.H{
		"TransactionID": transaction.ID,
		"StationName":   station.Name,
		"IsLoggedIn":    true,
	})
}

// ReopenRentalDoor allows a user to trigger the lock again after the initial rental success.
func (h *RentalHandler) ReopenRentalDoor(c *gin.Context) {
	txID, err := strconv.Atoi(c.PostForm("transaction_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	var transaction models.Transaction
	if err := h.DB.Preload("PowerbankStationOrigin").
		Where("id = ? AND user_id = ? AND status = ?", txID, userID, "Ongoing").
		First(&transaction).Error; err != nil {
		c.String(http.StatusNotFound, "Active rental transaction not found")
		return
	}

	// Trigger the lock again
	esp32.TriggerLock(transaction.PowerbankStationOrigin.IPAddress, "open")

	// Redirect back to the success page with a confirmation message
	c.Redirect(http.StatusFound, "/rental/success/"+strconv.Itoa(txID))
}

// ReopenReturnDoor allows a user to trigger the lock again during the return process.
func (h *RentalHandler) ReopenReturnDoor(c *gin.Context) {
	txID, err := strconv.Atoi(c.PostForm("transaction_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	var transaction models.Transaction
	if err := h.DB.Preload("PowerbankStationReturn").
		Where("id = ? AND user_id = ? AND status = ?", txID, userID, "Returned").
		First(&transaction).Error; err != nil {
		c.String(http.StatusNotFound, "Return transaction not found")
		return
	}

	// Trigger the lock again
	esp32.TriggerLock(transaction.PowerbankStationReturn.IPAddress, "open")

	// Redirect back to the return success page
	c.Redirect(http.StatusFound, "/return/success/"+strconv.Itoa(txID))
}
