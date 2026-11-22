package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/shravastee-thakur/Go-Auth/internal/config"
	"github.com/shravastee-thakur/Go-Auth/internal/middleware"
	"github.com/shravastee-thakur/Go-Auth/internal/models"
	"github.com/shravastee-thakur/Go-Auth/internal/utils"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type AuthController struct {
	UserCollection *mongo.Collection
	Config         *config.Config
}

// NewAuthController is like a constructor in JS
func NewAuthController(db *mongo.Database, cfg *config.Config) *AuthController {
	return &AuthController{
		UserCollection: db.Collection("users"),
		Config:         cfg,
	}
}

// POST /api/v1/auth/register
func (ac *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	// 1. Parse request body (like const { name, email, password } = req.body)
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		utils.WriteError(w, http.StatusBadRequest, "Name, email, and password are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 2. Check if email already exists (like User.findOne({ email }))
	var existing models.User
	err := ac.UserCollection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing)
	if err == nil {
		utils.WriteError(w, http.StatusConflict, "Email already registered")
		return
	}
	if err != mongo.ErrNoDocuments {
		utils.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// 3. Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// 4. Create user document
	now := time.Now()
	newUser := models.User{
		Id:        bson.NewObjectID(),
		Name:      req.Name,
		Email:     req.Email,
		Password:  hashedPassword,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := ac.UserCollection.InsertOne(ctx, newUser); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{
		"message": "User registered successfully",
		"user":    newUser, // Password & RefreshToken are hidden thanks to json:"-"
	})
}

// POST /api/v1/auth/login
func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 1. Find user by email
	var user models.User
	err := ac.UserCollection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// 2. Verify password
	if !utils.CheckPassword(req.Password, user.Password) {
		utils.WriteError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// 3. Generate Access & Refresh tokens
	accessToken, refreshToken, err := utils.GenerateTokens(
		user.Id.Hex(),
		user.Email,
		ac.Config.AccessSecret,
		ac.Config.RefreshSecret,
	)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// 4. Save Refresh Token in DB (like User.updateOne({ _id }, { $set: { refresh_token } }))
	_, err = ac.UserCollection.UpdateOne(
		ctx,
		bson.M{"_id": user.Id},
		bson.M{"$set": bson.M{"refresh_token": refreshToken, "updated_at": time.Now()}},
	)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to save session")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// POST /api/v1/auth/refresh
func (ac *AuthController) Refresh(w http.ResponseWriter, r *http.Request)  {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == ""{
		utils.WriteError(w, http.StatusBadRequest, "Refresh token is required")
		return
	}

	// 1. Verify the refresh token signature & expiry
	claims, err := utils.ValidateToken(req.RefreshToken, ac.Config.RefreshSecret)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 2. Check that the user exists and the token matches what's stored in DB
	objID, err := bson.ObjectIDFromHex(claims.UserID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, "Invalid user ID")
		return
	}

	var user models.User
	err = ac.UserCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil || user.RefreshToken != req.RefreshToken {
		utils.WriteError(w, http.StatusUnauthorized, "Refresh token revoked or invalid")
		return
	}

	// 3. Rotate: issue a fresh pair of tokens and save the new refresh token
	newAccess, newRefresh, err := utils.GenerateTokens(
		user.Id.Hex(),
		user.Email,
		ac.Config.AccessSecret,
		ac.Config.RefreshSecret,
	)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	_, err = ac.UserCollection.UpdateOne(
		ctx,
		bson.M{"_id": user.Id},
		bson.M{"$set": bson.M{"refresh_token": newRefresh, "updated_at": time.Now()}},
	)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update session")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"access_token":  newAccess,
		"refresh_token": newRefresh,
	})
	
}

// GET /api/v1/auth/profile (Protected route)
func (ac *AuthController) Profile(w http.ResponseWriter, r *http.Request) {
	// Read user claims from context (like const user = req.user)
	claims, ok := r.Context().Value(middleware.UserContextKet).(*utils.Claims)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id": claims.UserID,
		"email":   claims.Email,
	})
}