package main

import (
	"kbt-cuy/handlers"
	"kbt-cuy/models"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. Database Connection (Local SQLite)
	db, err := gorm.Open(sqlite.Open("powerbank.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 2. Migrate Schema
	db.AutoMigrate(&models.User{}, &models.PowerbankStation{}, &models.Powerbank{}, &models.Transaction{})

	// 3. Seed Demo Data
	seedData(db)

	// 4. Router Setup
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	authHandler := &handlers.AuthHandler{DB: db}
	rentalHandler := &handlers.RentalHandler{DB: db}

	// 5. Routes
	r.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		isLoggedIn := session.Get("user_id") != nil
		c.HTML(http.StatusOK, "index.html", gin.H{"IsLoggedIn": isLoggedIn})
	})

	r.GET("/login", func(c *gin.Context) { c.HTML(http.StatusOK, "login.html", nil) })
	r.POST("/login", authHandler.Login)
	r.GET("/register", func(c *gin.Context) { c.HTML(http.StatusOK, "register.html", nil) })
	r.POST("/register", authHandler.Register)
	r.GET("/logout", authHandler.Logout)

	// Protected Routes
	authorized := r.Group("/")
	authorized.Use(AuthRequired())
	{
		authorized.GET("/account", authHandler.Account)

		// Rental Flow
		authorized.GET("/rental", rentalHandler.ShowRentalStations)
		authorized.GET("/rental/:id/pay", rentalHandler.ShowPaymentPage)
		authorized.POST("/rental/process", rentalHandler.RentPowerbank)
		authorized.POST("/rental/retry-open", rentalHandler.RetryOpenDoor)

		// Return Flow
		authorized.GET("/return", rentalHandler.ShowReturnStations)
		authorized.POST("/return", rentalHandler.ReturnPowerbank)
		authorized.POST("/return/retry-open", rentalHandler.RetryReturnOpenDoor) // New Route
	}

	r.Run(":8080")
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user_id")
		if user == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func seedData(db *gorm.DB) {
	var count int64
	db.Model(&models.PowerbankStation{}).Count(&count)
	if count == 0 {
		station := models.PowerbankStation{
			Name: "Central Station", Latitude: -6.2, Longitude: 106.8, Capacity: 10, PowerbankLeft: 2, IPAddress: "192.168.1.50",
		}
		db.Create(&station)

		db.Create(&models.Powerbank{PowerbankCode: "PB-001", Capacity: 10000, Status: "Available", CurrentStationID: &station.ID})
		db.Create(&models.Powerbank{PowerbankCode: "PB-002", Capacity: 10000, Status: "Available", CurrentStationID: &station.ID})
	}
}
