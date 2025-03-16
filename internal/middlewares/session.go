package middlewares

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/momokii/echo-notes/internal/databases"

	sso_models "github.com/momokii/go-sso-web/pkg/models"
	sessionRepo "github.com/momokii/go-sso-web/pkg/repository/session"
	sso_user "github.com/momokii/go-sso-web/pkg/repository/user"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

var (
	Store   *session.Store
	SSO_URL = os.Getenv("SSO_URL")
)

type SessionMiddleware struct {
	dbService   databases.DBService
	userRepo    sso_user.UserRepo
	sessionRepo sessionRepo.SessionRepo
}

func NewSessionMiddleware(
	dbService databases.DBService,
	userRepo sso_user.UserRepo,
	sessionRepo sessionRepo.SessionRepo,
) *SessionMiddleware {

	Store = session.New(session.Config{
		Expiration:     7 * time.Hour,
		CookieSecure:   true,
		CookieHTTPOnly: true,
		CookieName:     "session_id_echonotes",
		KeyLookup:      "cookie:session_id_echonotes",
	})

	log.Println("Session store initialized")

	return &SessionMiddleware{
		dbService:   dbService,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func CreateSession(c *fiber.Ctx, key string, value interface{}) error {
	sess, err := Store.Get(c)
	if err != nil {
		return err
	}
	defer sess.Save()

	sess.Set(key, value)

	return nil
}

func DeleteSession(c *fiber.Ctx) error {
	sess, err := Store.Get(c)
	if err != nil {
		return err
	}
	defer sess.Save()

	sess.Destroy()

	return nil
}

func CheckSession(c *fiber.Ctx, key string) (interface{}, error) {
	sess, err := Store.Get(c)
	if err != nil {
		return nil, err
	}

	return sess.Get(key), nil
}

// IsNotAuth middleware for non-authenticated routes
func (m *SessionMiddleware) IsNotAuth(c *fiber.Ctx) error {
	userid, err := CheckSession(c, "id")
	if err != nil {
		DeleteSession(c)
		return c.Redirect(SSO_URL)
	}

	session_id, err := CheckSession(c, "session_id")
	if err != nil {
		DeleteSession(c)
		return c.Redirect(SSO_URL)
	}

	if userid != nil && session_id != nil {
		return c.Redirect("/")
	}

	return c.Next()
}

// IsAuth middleware for authenticated routes
func (m *SessionMiddleware) IsAuth(c *fiber.Ctx) error {
	userid, err := CheckSession(c, "id")
	if err != nil {
		DeleteSession(c)
		return c.Redirect(SSO_URL)
	}

	session_id, err := CheckSession(c, "session_id")
	if err != nil {
		DeleteSession(c)
		return c.Redirect(SSO_URL)
	}

	if userid == nil || session_id == nil {
		DeleteSession(c)
		return c.Redirect(SSO_URL)
	}

	var userSession sso_models.UserSession

	err, _ = m.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {
		sessData, err := m.sessionRepo.FindSession(tx, session_id.(string), userid.(int))
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		if sessData.Id == 0 && sessData.UserId == 0 && sessData.SessionId == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Session not found"), fiber.StatusUnauthorized
		}

		userData, err := m.userRepo.FindByID(tx, userid.(int))
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		userSession = sso_models.UserSession{
			Id:               userData.Id,
			Username:         userData.Username,
			CreditToken:      userData.CreditToken,
			LastFirstLLMUsed: userData.LastFirstLLMUsed,
		}

		return nil, fiber.StatusOK
	})

	if err != nil {
		DeleteSession(c)
		return c.Redirect(SSO_URL)
	}

	c.Locals("user", userSession)

	return c.Next()
}
