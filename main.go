package main

import (
	"database/sql"
	"kbt-cuy/config"
	"kbt-cuy/handlers"
	"kbt-cuy/models"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. Load Configuration
	config.LoadConfig()

	// 2. Database Connection (Turso)
	sqlDB, err := sql.Open("libsql", config.TursoURL+"?authToken="+config.TursoToken)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to initialize GORM:", err)
	}

	// 3. Migrate Schema
	db.AutoMigrate(&models.User{}, &models.PowerbankStation{}, &models.Powerbank{}, &models.Transaction{})

	// 4. Seed Demo Data
	seedData(db)

	// 5. Router Setup
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	authHandler := &handlers.AuthHandler{DB: db}
	rentalHandler := &handlers.RentalHandler{DB: db}
	paymentHandler := &handlers.PaymentHandler{DB: db, Core: config.MidtransCore, Snap: config.MidtransSnap}
	mapHandler := &handlers.MapHandler{DB: db}

	// 6. Routes
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

		// Map View
		authorized.GET("/map", mapHandler.ShowMap)

		// Rental Flow
		authorized.GET("/rental", rentalHandler.ShowRentalStations)
		authorized.GET("/rental/:id/pay", paymentHandler.ShowPaymentPage)
		authorized.POST("/payment/create", paymentHandler.CreateTransaction)
		authorized.GET("/payment/status/:id", paymentHandler.GetPaymentStatus)

		authorized.GET("/rental/success/:id", rentalHandler.RentalSuccess)
		authorized.POST("/rental/re-open", rentalHandler.ReopenRentalDoor)

		// Return Flow
		authorized.GET("/return", rentalHandler.ShowReturnStations)
		authorized.POST("/return", rentalHandler.ReturnPowerbank)
		authorized.POST("/return/re-open", rentalHandler.ReopenReturnDoor)
	}

	r.POST("/payment/notification", paymentHandler.PaymentNotification)
	r.Run(":8085")
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
		// 1. Create Main Central Station
		station1 := models.PowerbankStation{
			Name: "Kantin Pusat ITS", Latitude: -7.2839100, Longitude: 112.7940321, Capacity: 10, PowerbankLeft: 5, IPAddress: "192.168.1.50",
		}
		db.Create(&station1)
		db.Create(&models.Powerbank{PowerbankCode: "PB-001", Capacity: 10000, Status: "Available", CurrentStationID: &station1.ID})
		db.Create(&models.Powerbank{PowerbankCode: "PB-002", Capacity: 10000, Status: "Available", CurrentStationID: &station1.ID})

		station2 := models.PowerbankStation{
			Name: "Tower 2 ITS", Latitude: -7.2851831, Longitude: 112.7952606, Capacity: 8, PowerbankLeft: 3, IPAddress: "192.168.1.50",
		}
		db.Create(&station2)
		db.Create(&models.Powerbank{PowerbankCode: "PB-003", Capacity: 10000, Status: "Available", CurrentStationID: &station2.ID})
	}
}
