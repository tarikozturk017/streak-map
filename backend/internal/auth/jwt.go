package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tarikozturk017/streak-map/backend/internal/models"
)

type JWTService struct {
	secretKey         []byte
	accessTokenTTL    time.Duration
	refreshTokenTTL   time.Duration
}

func NewJWTService(secretKey string, accessTTL, refreshTTL time.Duration) *JWTService {
	return &JWTService{
		secretKey:         []byte(secretKey),
		accessTokenTTL:    accessTTL,
		refreshTokenTTL:   refreshTTL,
	}
}

func (j *JWTService) GenerateTokenPair(user *models.User) (*models.TokenPair, error) {
	accessToken, err := j.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := j.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(j.accessTokenTTL.Seconds()),
	}, nil
}

func (j *JWTService) generateAccessToken(user *models.User) (string, error) {
	claims := &models.JWTClaims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Type:     "access",
		ExpiresAt: time.Now().Add(j.accessTokenTTL).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  claims.UserID,
		"email":    claims.Email,
		"username": claims.Username,
		"type":     claims.Type,
		"exp":      claims.ExpiresAt,
		"iat":      claims.IssuedAt,
	})

	return token.SignedString(j.secretKey)
}

func (j *JWTService) generateRefreshToken(user *models.User) (string, error) {
	claims := &models.JWTClaims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Type:     "refresh",
		ExpiresAt: time.Now().Add(j.refreshTokenTTL).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  claims.UserID,
		"email":    claims.Email,
		"username": claims.Username,
		"type":     claims.Type,
		"exp":      claims.ExpiresAt,
		"iat":      claims.IssuedAt,
	})

	return token.SignedString(j.secretKey)
}

func (j *JWTService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid user_id in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user_id format")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, errors.New("invalid email in token")
	}

	username, ok := claims["username"].(string)
	if !ok {
		return nil, errors.New("invalid username in token")
	}

	tokenType, ok := claims["type"].(string)
	if !ok {
		return nil, errors.New("invalid type in token")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid exp in token")
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, errors.New("invalid iat in token")
	}

	return &models.JWTClaims{
		UserID:   userID,
		Email:    email,
		Username: username,
		Type:     tokenType,
		ExpiresAt: int64(exp),
		IssuedAt:  int64(iat),
	}, nil
}