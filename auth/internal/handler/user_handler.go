package handler

import (
	"errors"
	"linkv-auth/internal/response"
	"linkv-auth/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

type UserRegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register godoc
//
//	@Summary		Регистрация пользователя
//	@Description	Регистрация нового пользователя
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		UserRegisterRequest				true	"Параметры регистрации пользователя"
//	@Success		201		{object}	response.UserRegisterResponse	"Успешная регистрация пользователя"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка валидации"
//	@Failure		409		{object}	response.ErrorResponse			"Пользователь уже существует"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/api/v1/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "Ошибка валидации"})
		return
	}

	user, err := h.service.Register(req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			c.JSON(http.StatusConflict, response.ErrorResponse{Error: "Пользователь уже существует"})
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Error: "Ошибка сервера"})
		}
		return
	}

	resp := response.UserRegisterResponse{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}

	c.JSON(http.StatusCreated, resp)
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Login godoc
//
//	@Summary		Вход пользователя
//	@Description	Вход существующего пользователя
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		LoginRequest			true	"Параметры входа пользователя"
//	@Success		200		{object}	response.TokenResponse	"Успешный вход пользователя"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка валидации"
//	@Failure		404		{object}	response.ErrorResponse	"Пользователь не найден"
//	@Failure		401		{object}	response.ErrorResponse	"Неверный пароль"
//	@Failure		500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/api/v1/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "Ошибка валидации"})
		return
	}

	access, refresh, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, response.ErrorResponse{Error: "Пользователь не найден"})
		case errors.Is(err, service.ErrInvalidPassword):
			c.JSON(http.StatusUnauthorized, response.ErrorResponse{Error: "Неверный пароль"})
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Error: "Ошибка сервера"})
		}
		return
	}

	resp := response.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}

	c.JSON(http.StatusOK, resp)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh godoc
//
//	@Summary		Обновление токена
//	@Description	Обновление refresh-токена
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		RefreshRequest			true	"Параметры обновления токена"
//	@Success		200		{object}	response.TokenResponse	"Успешное обновление токена"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка валидации"
//	@Failure		401		{object}	response.ErrorResponse	"Неверный токен"
//	@Failure		500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/api/v1/auth/refresh [post]
func (h *UserHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "Ошибка валидации"})
		return
	}

	access, refresh, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			c.JSON(http.StatusUnauthorized, response.ErrorResponse{Error: "Неверный токен"})
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Error: "Ошибка сервера"})
		}
		return
	}

	resp := response.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}

	c.JSON(http.StatusOK, resp)
}

// Profile godoc
//
//	@Summary		Получпение профиля
//	@Description	Получение своего профиля по токену
//	@Security		BearerAuth
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.UserResponse	"Полученный профиль"
//	@Failure		404	{object}	response.ErrorResponse	"Пользователь не найден"
//	@Router			/api/v1/profile/me [get]
func (h *UserHandler) Profile(c *gin.Context) {
	var userID uuid.UUID
	if val, exists := c.Get("user_id"); exists {
		if parsed, err := uuid.Parse(val.(string)); err == nil {
			userID = parsed
		} else {
			c.JSON(http.StatusUnauthorized, response.ErrorResponse{Error: "Invalid user ID"})
			return
		}
	} else {
		c.JSON(http.StatusUnauthorized, response.ErrorResponse{Error: "Missing user ID"})
		return
	}

	user, err := h.service.Profile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, response.ErrorResponse{Error: "Пользователь не найден"})
		return
	}

	userResponse := response.UserResponse{
		Name:  user.Name,
		Email: user.Email,
	}

	c.JSON(http.StatusOK, userResponse)
}
