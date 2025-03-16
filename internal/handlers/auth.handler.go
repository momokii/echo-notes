package handlers

import (
	"database/sql"
	"errors"
	"os"

	"github.com/momokii/echo-notes/internal/databases"
	"github.com/momokii/echo-notes/internal/middlewares"
	"github.com/momokii/echo-notes/pkg/utils"

	sso_session "github.com/momokii/go-sso-web/pkg/repository/session"
	sso_user "github.com/momokii/go-sso-web/pkg/repository/user"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthHandler struct {
	userRepo    sso_user.UserRepo
	sessionRepo sso_session.SessionRepo
	dbService   databases.DBService
}

func NewAuthHandler(userRepo sso_user.UserRepo, sessionRepo sso_session.SessionRepo, dbService databases.DBService) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		dbService:   dbService,
	}
}

// SSO func
func (h *AuthHandler) SSOAuthLogin(c *fiber.Ctx) error {
	// get jwt token from request
	token := c.Query("token")
	if token == "" {
		return errors.New("token is required")
	}

	// validate token
	token_data, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return errors.New("invalid token")
	}

	session_id := token_data.Claims.(jwt.MapClaims)["session_id"].(string)
	user_id := int(token_data.Claims.(jwt.MapClaims)["user_id"].(float64))

	err, statusCode := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		session_check, err := h.sessionRepo.FindSession(tx, session_id, user_id)
		if err != nil {
			return errors.New("Internal server error on find session: " + err.Error()), fiber.StatusInternalServerError
		}

		if session_check.Id == 0 && session_check.SessionId == "" && session_check.UserId == 0 {

			return errors.New("invalid, session not found"), fiber.StatusUnauthorized
		}

		// save session to fiber session data
		if err := middlewares.CreateSession(c, "id", user_id); err != nil {
			return errors.New("Internal server error on create session: " + err.Error()), fiber.StatusInternalServerError
		}

		if err := middlewares.CreateSession(c, "session_id", session_id); err != nil {
			return errors.New("Internal server error on create session: " + err.Error()), fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	})

	if err != nil {
		return utils.ResponseError(c, statusCode, err.Error())
	}

	return c.Redirect("/")
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// delete session here
	middlewares.DeleteSession(c)

	return utils.ResponseMessage(c, fiber.StatusOK, "Logout success")
}
