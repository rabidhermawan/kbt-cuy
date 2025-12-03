package handlers

import (
	"kbt-cuy/models"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB *gorm.DB
}

func (h *AuthHandler) Register(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := models.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
	}

	if result := h.DB.Create(&user); result.Error != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"Error": "Username or Email already exists"})
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	var user models.User
	if err := h.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Error": "Invalid credentials"})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Save()

	c.Redirect(http.StatusFound, "/account")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Account(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	var user models.User
	// Preload transactions and the powerbank details
	h.DB.Preload("Transactions.Powerbank").
		Preload("Transactions.PowerbankStationOrigin").
		Preload("Transactions.PowerbankStationReturn").
		First(&user, userID)

	c.HTML(http.StatusOK, "account.html", gin.H{
		"User":       user,
		"IsLoggedIn": true,
	})
}
