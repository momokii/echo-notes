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
	meeting_summaries "github.com/momokii/echo-notes/internal/repository/meeting-summaries"
	"github.com/momokii/echo-notes/pkg/utils"
	"github.com/momokii/go-llmbridge/pkg/openai"
)

type SummariesHandler struct {
	openaiClient         openai.OpenAI
	dbService            databases.DBService
	userRepo             sso_user.UserRepo
	meetingSummariesRepo meeting_summaries.MeetingSummaries
}

func NewSummariesHandler(
	openaiClient openai.OpenAI,
	dbService databases.DBService,
	userRepo sso_user.UserRepo,
	meetingSummariesRepo meeting_summaries.MeetingSummaries,
) *SummariesHandler {
	return &SummariesHandler{
		openaiClient:         openaiClient,
		dbService:            dbService,
		userRepo:             userRepo,
		meetingSummariesRepo: meetingSummariesRepo,
	}
}

func (h *SummariesHandler) SummariesView(c *fiber.Ctx) error {
	user := c.Locals("user").(sso_models.UserSession)

	return c.Render("summaries", fiber.Map{
		"Title": "Echo Notes - My Summaries",
		"User":  user,
	})
}

func (h *SummariesHandler) RecorderView(c *fiber.Ctx) error {
	user := c.Locals("user").(sso_models.UserSession)

	return c.Render("dashboard", fiber.Map{
		"Title": "Echo Notes",
		"User":  user,
	})
}

func (h *SummariesHandler) SummariesReduceUserToken(c *fiber.Ctx) error {

	// get user from session
	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	const FEATURE_COST = utils.MEETING_SUMMARY_AI_COST

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
		if user.CreditToken < FEATURE_COST {
			return fmt.Errorf("Not enough credit token to use this feature"), fiber.StatusBadRequest
		}

		// for now, just reduce the token
		if err := sso_utils.UpdateUserCredit(tx, h.userRepo, user, FEATURE_COST); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success reduce user token", fiber.Map{
		"feature_cost":     FEATURE_COST,
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

	system_prompt := utils.SYSTEM_PROMPT_RECORDING_SUMMARIES

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

func (h *SummariesHandler) SaveSummaries(c *fiber.Ctx) error {

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	newSummariesData := new(models.MeetingSummariesCreate)
	if err := c.BodyParser(newSummariesData); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "failed to parse request body")
	}

	// parsing user id to new summaries data
	newSummariesData.UserId = user_session.Id

	// TODO: setup validator here later hehe

	// start transaction
	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		if err := h.meetingSummariesRepo.Create(tx, *newSummariesData); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "success save summaries data")
}

func (h *SummariesHandler) GetSummaries(c *fiber.Ctx) error {
	var summariesDataReturn []models.MeetingSummaries
	total_data := 0

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 10)
	search := c.Query("search")
	order_by := c.Query("order_by")

	paginationData := models.PaginationFiltering{
		Page:    page,
		PerPage: perPage,
		Search:  search,
		OrderBy: order_by,
	}

	err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		summariesData, total, err := h.meetingSummariesRepo.Find(tx, paginationData, user_session.Id)
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		total_data = total
		summariesDataReturn = *summariesData

		// if data is nil, return empty array so the response will be an empty array and not null
		if summariesDataReturn == nil {
			summariesDataReturn = []models.MeetingSummaries{}
		}

		return nil, fiber.StatusOK
	})

	if err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success get summaries data", fiber.Map{
		"summaries": summariesDataReturn,
		"pagination": fiber.Map{
			"total_page":   total_data/perPage + 1,
			"total_data":   total_data,
			"current_page": page,
			"per_page":     perPage,
		},
	})
}

func (h *SummariesHandler) GetOneSummary(c *fiber.Ctx) error {
	var summaryData models.MeetingSummaries

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id_summary := c.Params("id")
	id_summary_int, err := strconv.Atoi(id_summary)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "invalid id")
	}

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		data, err := h.meetingSummariesRepo.FindById(tx, id_summary_int, user_session.Id)

		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("data not found"), fiber.StatusNotFound
			}

			return err, fiber.StatusInternalServerError
		}

		summaryData = *data

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success get one summaries data", fiber.Map{
		"summary_data": summaryData,
	})
}

func (h *SummariesHandler) EditSummaries(c *fiber.Ctx) error {

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// get id summary data need to edit
	id_summary := c.Params("id")
	id_summary_int, err := strconv.Atoi(id_summary)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "invalid id")
	}

	// parsing req data body
	new_summary_data := new(models.MeetingSummariesUpdate)
	if err := c.BodyParser(new_summary_data); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "failed to parse request body")
	}

	new_summary_data.UserId = user_session.Id

	// TODO: setup validator here later hehe

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		// check if data exist
		data_summary, err := h.meetingSummariesRepo.FindById(tx, id_summary_int, user_session.Id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("data not found"), fiber.StatusNotFound
			}

			return err, fiber.StatusInternalServerError
		}

		// replace data with new data (only data can edit is name and description)
		data_summary.Name = new_summary_data.Name
		data_summary.Description = new_summary_data.Description

		if err := h.meetingSummariesRepo.Update(tx, *data_summary); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "success edit one summaries data")
}

func (h *SummariesHandler) DeleteSummaries(c *fiber.Ctx) error {

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	deleted_id := c.Params("id")
	deleted_id_int, err := strconv.Atoi(deleted_id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "invalid id")
	}

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		// check if data exist
		summary_data, err := h.meetingSummariesRepo.FindById(tx, deleted_id_int, user_session.Id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Data not found"), fiber.StatusNotFound
			}

			return err, fiber.StatusInternalServerError
		}

		if err := h.meetingSummariesRepo.Delete(tx, *summary_data); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "success delete one summaries data")
}
