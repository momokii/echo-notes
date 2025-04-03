package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/momokii/echo-notes/internal/databases"
	"github.com/momokii/echo-notes/internal/handlers"
	"github.com/momokii/echo-notes/internal/middlewares"
	meeting_group_summaries "github.com/momokii/echo-notes/internal/repository/meeting-group-summaries"
	meeting_summaries "github.com/momokii/echo-notes/internal/repository/meeting-summaries"

	"github.com/momokii/go-llmbridge/pkg/openai"

	sso_session "github.com/momokii/go-sso-web/pkg/repository/session"
	sso_user "github.com/momokii/go-sso-web/pkg/repository/user"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	DEVMODE := os.Getenv("APP_ENV")
	PORT := os.Getenv("PORT")

	// OpenAI client init
	openaiClient, err := openai.New(
		os.Getenv("OPENAI_API_KEY"),
		"",
		"",
		openai.WithModel(os.Getenv("OPENAI_MODEL_NAME")),
	)
	if err != nil {
		log.Println("Error creating OpenAI client: ", err)
		return
	} else {
		log.Println("OpenAI client initialized...")
	}

	// init database and session
	postgreService := databases.NewPostgresService()
	sessionService := middlewares.NewSessionMiddleware(postgreService, *sso_user.NewUserRepo(), *sso_session.NewSessionRepo())

	// repo init
	userRepo := sso_user.NewUserRepo()
	sessionRepo := sso_session.NewSessionRepo()
	meetingSummariesRepo := meeting_summaries.NewMeetingSummaries()
	meetingGroupSummariesRepo := meeting_group_summaries.NewMeetingGroupSummaries()

	// handler init
	summariesHandler := handlers.NewSummariesHandler(openaiClient, postgreService, *userRepo, *meetingSummariesRepo)
	summariesGroupHandler := handlers.NewSummariesGroupHandler(openaiClient, postgreService, *userRepo, *meetingGroupSummariesRepo, *meetingSummariesRepo)
	authHandler := handlers.NewAuthHandler(*userRepo, *sessionRepo, postgreService)

	engine := html.New("./web", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).Render("error", fiber.Map{
				"Code":    code,
				"Message": err.Error(),
			})
		},
	})

	api := app.Group("/api")

	app.Use(cors.New())
	app.Use(logger.New())
	app.Use(helmet.New())
	// production only for using recover for panic resolver
	if DEVMODE == "production" {
		app.Use(recover.New())
	}
	app.Static("/web", "./web")

	// auth sso
	app.Get("/auth/sso", sessionService.IsNotAuth, authHandler.SSOAuthLogin)
	api.Post("/logout", sessionService.IsAuth, authHandler.Logout)

	// view/ pages routing
	app.Get("/", sessionService.IsAuth, summariesHandler.RecorderView)
	app.Get("/summaries", sessionService.IsAuth, summariesHandler.SummariesView)

	// audio recorder pages api
	api.Post("/audio/chunks", sessionService.IsAuth, summariesHandler.ProcessChunkAudio)
	api.Post("/audio/summaries", sessionService.IsAuth, summariesHandler.SummariesData)
	api.Post("/audio/summaries/cost", sessionService.IsAuth, summariesHandler.SummariesReduceUserToken)

	// summaries api
	api.Get("/summaries", sessionService.IsAuth, summariesHandler.GetSummaries)
	api.Get("/summaries/:id", sessionService.IsAuth, summariesHandler.GetOneSummary)
	api.Post("/summaries", sessionService.IsAuth, summariesHandler.SaveSummaries)
	api.Patch("/summaries/:id", sessionService.IsAuth, summariesHandler.EditSummaries)
	api.Delete("/summaries/:id", sessionService.IsAuth, summariesHandler.DeleteSummaries)

	// grouping summaries api
	api.Get("/summaries/groups", sessionService.IsAuth, summariesGroupHandler.GetGroupSummaries)
	api.Post("/summaries/groups/cost", sessionService.IsAuth, summariesGroupHandler.SummariesGroupReduceUserToken)
	api.Get("/summaries/groups/:id", sessionService.IsAuth, summariesGroupHandler.GetOneGroupSummaries)
	api.Post("/summaries/groups/llm", sessionService.IsAuth, summariesGroupHandler.CreateGroupSummariesDataLLM)
	api.Post("/summaries/groups", sessionService.IsAuth, summariesGroupHandler.SaveGroupSummaries)
	api.Patch("/summaries/groups/:id", sessionService.IsAuth, summariesGroupHandler.UpdateGroupSummaries)
	api.Delete("/summaries/groups/:id", sessionService.IsAuth, summariesGroupHandler.DeleteGroupSummaries)

	if DEVMODE != "development" && DEVMODE != "production" {
		log.Println("APP_ENV not set")
	} else {
		log.Println("Running on: " + DEVMODE)
		if err := app.Listen(":" + PORT); err != nil {
			log.Println("Error running Server: ", err)
		}
	}
}
