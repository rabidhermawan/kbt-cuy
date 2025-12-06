package handlers

import (
	"kbt-cuy/esp32"
	"kbt-cuy/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	DB   *gorm.DB
	Core *coreapi.Client
	Snap *snap.Client
}

// ShowPaymentPage renders the payment confirmation screen
func (h *PaymentHandler) ShowPaymentPage(c *gin.Context) {
	stationID := c.Param("id")
	var station models.PowerbankStation
	if err := h.DB.First(&station, stationID).Error; err != nil {
		c.String(http.StatusNotFound, "Station not found")
		return
	}

	clientKey := os.Getenv("MIDTRANS_CLIENT_KEY_FRONTEND")

	c.HTML(http.StatusOK, "payment.html", gin.H{
		"Station":    station,
		"IsLoggedIn": true,
		"ClientKey":  clientKey,
	})
}

// CreateTransaction creates a new Midtrans Snap transaction
func (h *PaymentHandler) CreateTransaction(c *gin.Context) {
	stationIDStr := c.PostForm("station_id")
	stationID, _ := strconv.ParseUint(stationIDStr, 10, 32)
	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	var user models.User
	h.DB.First(&user, userID)

	orderID := "ORDER-" + strconv.FormatInt(time.Now().Unix(), 10)

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: 10000,
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.Username,
			Email: user.Email,
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    stationIDStr,
				Name:  "Powerbank Rental",
				Price: 10000,
				Qty:   1,
			},
		},
	}

	snapResp, err := h.Snap.CreateTransaction(snapReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}

	transaction := models.Transaction{
		UserID:                   userID,
		PowerbankStationOriginID: uint(stationID),
		Status:                   "Pending",
		OrderID:                  orderID,
		PaymentToken:             snapResp.Token,
		PaymentRedirectURL:       snapResp.RedirectURL,
	}

	if result := h.DB.Create(&transaction); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":          snapResp.Token,
		"transaction_id": transaction.ID,
	})
}

// PaymentNotification handles the webhook from Midtrans
func (h *PaymentHandler) PaymentNotification(c *gin.Context) {
	var notificationPayload map[string]interface{}
	c.ShouldBindJSON(&notificationPayload)

	orderID, _ := notificationPayload["order_id"].(string)

	transactionStatus, err := h.Core.CheckTransaction(orderID)
	if err != nil {
		return
	}

	if transactionStatus != nil {
		if transactionStatus.TransactionStatus == "capture" || transactionStatus.TransactionStatus == "settlement" {
			h.processSuccessfulRental(orderID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetPaymentStatus polls Midtrans for the latest transaction status
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	txID := c.Param("id")
	var transaction models.Transaction
	h.DB.First(&transaction, txID)

	transactionStatus, err := h.Core.CheckTransaction(transaction.OrderID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "pending"})
		return
	}

	if transactionStatus != nil {
		// Payment is confirmed by Midtrans
		if transactionStatus.TransactionStatus == "capture" || transactionStatus.TransactionStatus == "settlement" {
			// Check if we need to process the rental (state is still "Pending")
			if transaction.Status == "Pending" {
				h.processSuccessfulRental(transaction.OrderID)
				// After processing, let's check the new status of our internal transaction
				var updatedTx models.Transaction
				h.DB.First(&updatedTx, txID)
				if updatedTx.Status == "Ongoing" {
					c.JSON(http.StatusOK, gin.H{"status": "success", "transaction_id": updatedTx.ID})
				} else {
					// This can happen if processSuccessfulRental fails (e.g., no powerbanks left)
					c.JSON(http.StatusOK, gin.H{"status": "failed"})
				}
				return // Important to return here
			} else if transaction.Status == "Ongoing" {
				// Already processed, just confirm success
				c.JSON(http.StatusOK, gin.H{"status": "success", "transaction_id": transaction.ID})
				return
			}
		} else if transactionStatus.TransactionStatus == "deny" || transactionStatus.TransactionStatus == "expire" || transactionStatus.TransactionStatus == "cancel" {
			// Handle failed payment
			if transaction.Status != "Failed" {
				transaction.Status = "Failed"
				h.DB.Save(&transaction)
			}
			c.JSON(http.StatusOK, gin.H{"status": "failed"})
			return
		}
	}

	// If none of the above, the payment is still pending
	c.JSON(http.StatusOK, gin.H{"status": "pending"})
}

// processSuccessfulRental handles the logic for a successful rental
func (h *PaymentHandler) processSuccessfulRental(orderID string) {
	var tx models.Transaction
	if err := h.DB.Where("order_id = ?", orderID).First(&tx).Error; err != nil {
		return
	}

	if tx.Status == "Ongoing" {
		return
	}

	var pb models.Powerbank
	if err := h.DB.Where("current_station_id = ? AND status = ?", tx.PowerbankStationOriginID, "Available").First(&pb).Error; err != nil {
		tx.Status = "Failed"
		h.DB.Save(&tx)
		return
	}

	dbTx := h.DB.Begin()

	pbId := pb.ID
	tx.Status = "Ongoing"
	tx.PowerbankID = &pbId
	if err := dbTx.Save(&tx).Error; err != nil {
		dbTx.Rollback()
		return
	}

	pb.Status = "Rented"
	pb.CurrentStationID = nil
	if err := dbTx.Save(&pb).Error; err != nil {
		dbTx.Rollback()
		return
	}

	var station models.PowerbankStation
	if err := dbTx.First(&station, tx.PowerbankStationOriginID).Error; err != nil {
		dbTx.Rollback()
		return
	}
	station.PowerbankLeft--
	if err := dbTx.Save(&station).Error; err != nil {
		dbTx.Rollback()
		return
	}

	dbTx.Commit()

	go esp32.TriggerLock(station.IPAddress, "open")
}
