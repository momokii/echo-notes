package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/momokii/echo-notes/internal/handlers"
	"github.com/momokii/go-llmbridge/pkg/openai"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	DEVMODE := os.Getenv("APP_ENV")
	PORT := os.Getenv("PORT")

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

	// handler init
	summariesHandler := handlers.NewSummariesHandler(openaiClient)

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
	app.Static("/web", "./web")

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("dashboard", fiber.Map{
			"Title": "Echo Notes",
		})
	})

	api.Post("/audio/chunks", summariesHandler.ProcessChunkAudio)
	api.Post("/audio/summaries", summariesHandler.SummariesData)

	if DEVMODE != "development" && DEVMODE != "production" {
		log.Println("APP_ENV not set")
	} else {
		log.Println("Running on: " + DEVMODE)
		if err := app.Listen(":" + PORT); err != nil {
			log.Println("Error running Server: ", err)
		}
	}
}
