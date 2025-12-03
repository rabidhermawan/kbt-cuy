package main

import (
	"kbt-cuy/handlers"
	"kbt-cuy/models"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	// Switch import from postgres to sqlite
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. Database Connection (Local SQLite for prototyping)
	// This creates a file named 'powerbank.db' in the project root
	db, err := gorm.Open(sqlite.Open("powerbank.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 2. Migrate Schema
	db.AutoMigrate(&models.User{}, &models.PowerbankStation{}, &models.Powerbank{}, &models.Transaction{})

	// 3. Seed Demo Data (Optional - for testing)
	seedData(db)

	// 4. Router Setup
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")

	// Session Middleware
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	// Handlers
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
		authorized.GET("/rental", rentalHandler.ShowRentalStations)
		authorized.POST("/rental", rentalHandler.RentPowerbank)
		authorized.GET("/return", rentalHandler.ShowReturnStations)
		authorized.POST("/return", rentalHandler.ReturnPowerbank)
	}

	// Run on localhost:8080
	r.Run("127.0.0.1:8080")
}

// AuthRequired Middleware
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
