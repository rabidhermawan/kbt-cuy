package handlers

import (
	"encoding/json"
	"kbt-cuy/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MapHandler struct {
	DB *gorm.DB
}

// ShowMap renders the station map view
func (h *MapHandler) ShowMap(c *gin.Context) {
	var stations []models.PowerbankStation
	h.DB.Find(&stations)

	// Marshal stations to JSON to safely embed in the script tag
	stationsJSON, err := json.Marshal(stations)
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not serialize station data")
		return
	}

	c.HTML(http.StatusOK, "map.html", gin.H{
		"StationsJSON": string(stationsJSON),
		"IsLoggedIn":   true,
	})
}
