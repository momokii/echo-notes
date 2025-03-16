package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	sso_models "github.com/momokii/go-sso-web/pkg/models"
	sso_user "github.com/momokii/go-sso-web/pkg/repository/user"
	sso_utils "github.com/momokii/go-sso-web/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/momokii/echo-notes/internal/databases"
	"github.com/momokii/echo-notes/internal/models"
	"github.com/momokii/echo-notes/pkg/utils"
	"github.com/momokii/go-llmbridge/pkg/openai"
)

type SummariesHandler struct {
	openaiClient openai.OpenAI
	dbService    databases.DBService
	userRepo     sso_user.UserRepo
}

func NewSummariesHandler(openaiClient openai.OpenAI, dbService databases.DBService, userRepo sso_user.UserRepo) *SummariesHandler {
	return &SummariesHandler{
		openaiClient: openaiClient,
		dbService:    dbService,
		userRepo:     userRepo,
	}
}

func (h *SummariesHandler) SummariesView(c *fiber.Ctx) error {
	user := c.Locals("user").(sso_models.UserSession)

	return c.Render("dashboard", fiber.Map{
		"Title": "Echo Notes",
		"User":  user,
	})
}

// TODO: test this function
func (h *SummariesHandler) SummariesReduceUserToken(c *fiber.Ctx) error {

	// get user from session
	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// start transaction
	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		user, err := h.userRepo.FindByID(tx, user_session.Id)
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		if user.Id == 0 {
			return fmt.Errorf("user not found"), fiber.StatusNotFound
		}

		// check user current token
		if user.CreditToken < utils.MEETING_SUMMARY_AI_COST {
			return fmt.Errorf("Not enough credit token to use this feature"), fiber.StatusBadRequest
		}

		// for now, just reduce the token
		if err := sso_utils.UpdateUserCredit(tx, h.userRepo, user, utils.MEETING_SUMMARY_AI_COST); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success reduce user token", fiber.Map{
		"feature_cost":     utils.MEETING_SUMMARY_AI_COST,
		"new_credit_token": user_session.CreditToken - utils.MEETING_SUMMARY_AI_COST,
	})
}

func (h *SummariesHandler) ProcessChunkAudio(c *fiber.Ctx) error {
	// Parse the form data
	form, err := c.MultipartForm()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "failed to parse form data")
	}

	// Get the audio file from the form data
	chunkNumberStr := form.Value["chunkNumber"][0]
	chunk_number, err := strconv.Atoi(chunkNumberStr)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "invalid chunk number")
	}
	files := form.File["audio"]
	if len(files) == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "no audio file found in form data")
	}

	// Get the audio file from the form data
	audioFile := files[0]

	// create request for OpenAI Speech to Text
	req := openai.OATranscriptionDefaultReq{
		File:     audioFile,
		Filename: audioFile.Filename,
	}

	oaResp, err := h.openaiClient.OpenAISpeechToTextDefault(&req)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to process chunk audio: "+err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success process chunk", fiber.Map{
		"chunk_number":    chunk_number,
		"translated_text": oaResp.Text,
	})
}

func (h *SummariesHandler) SummariesData(c *fiber.Ctx) error {

	summaries_input := new(models.SummariesData)
	if err := c.BodyParser(summaries_input); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "failed to parse request body")
	}

	system_prompt := `
		You are provided with the full transcription of a meeting. Your task is to generate two separate summaries based on the transcript and output them in the following JSON format:

		{
			"tldr_summary": <string>,
			"comprehensive_summary": <string>
		}

		Important Instructions:
		- Before summarizing, evaluate the content of the transcript:
		- If the transcript is very short (under 30 seconds or fewer than 50 words) or contains minimal substantive content, respond with a simplified format:
		
		{
			"tldr_summary": "The audio is too brief to extract meaningful summary points. It contains only [brief description of content].",
			"comprehensive_summary": "The transcript is too short to provide a comprehensive summary. Original content: [include full transcript]"
		}
		
		- Only proceed with full summarization if the transcript contains sufficient content for meaningful summaries.

		Part 1: TLDR (Too Long; Didn't Read) Summary
		- Produce a brief, high-level summary (3-5 sentences) that captures the most critical points, key decisions, and primary action items.
		- Ensure it is concise, clear, and written in a formal tone.
		- If there are no clear decisions or action items in the transcript, explicitly state this.

		Part 2: Comprehensive Summary
		- Create an in-depth summary that includes:
		- An overview of the meeting agenda (if mentioned).
		- Detailed discussion points and context around the topics covered.
		- Specific decisions made, including any deadlines and assigned responsibilities.
		- Additional relevant insights to provide a full picture of the meeting's outcomes.
		- Format this summary using clear headings and bullet points where appropriate to enhance readability (use markdown format for this comprehensive summary section).
		- The length of this summary should be proportional to the length and complexity of the original transcript.

		Language Adaptation:
		- Identify the primary language used in the transcript.
		- Generate both summaries (TLDR and comprehensive) in the same language as the transcript.
		- If the transcript is in Indonesian, provide summaries in Indonesian.
		- If the transcript is in English, provide summaries in English.
		- For transcripts with mixed languages, use the predominant language for the summaries.
		- Maintain proper grammar, formatting, and style conventions specific to the identified language.

		Additional Language-Specific Instructions:
		- For Indonesian transcripts: Gunakan bahasa Indonesia formal dan hindari penggunaan kata serapan yang tidak perlu. Pastikan tata bahasa dan ejaan sesuai dengan kaidah Bahasa Indonesia yang baik dan benar.
		- For English transcripts: Use formal English and appropriate business terminology. Ensure proper grammar and spelling according to standard English conventions.

		Both parts should maintain a formal and structured style, ensuring clarity and completeness. The TLDR provides a quick reference, while the comprehensive summary offers the full details needed for thorough understanding.

		**Important:** Ensure that your output is strictly in the provided JSON format with the keys "tldr_summary" and "comprehensive_summary".
	`

	messages := []openai.OAMessageReq{
		{
			Role:    "system",
			Content: system_prompt,
		},
		{
			Role:    "user",
			Content: summaries_input.FullTranslatedText,
		},
	}

	response_format := openai.OACreateResponseFormat(
		"summaries_response_format",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tldr_summary":          map[string]interface{}{"type": "string"},
				"comprehensive_summary": map[string]interface{}{"type": "string"},
			},
		},
	)

	response, err := h.openaiClient.OpenAIGetFirstContentDataResp(&messages, true, &response_format, false, nil)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to get response from OpenAI")
	}

	response_data := new(models.SummarizeLLMResponse)
	if err := json.Unmarshal([]byte(response.Content), &response_data); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to parse response body")
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success summarize data", fiber.Map{
		"summaries_data": response_data,
	})
}
