package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/momokii/echo-notes/internal/databases"
	"github.com/momokii/echo-notes/internal/models"
	"github.com/momokii/echo-notes/pkg/utils"
	"github.com/momokii/go-llmbridge/pkg/openai"

	meeting_group_summaries "github.com/momokii/echo-notes/internal/repository/meeting-group-summaries"
	meeting_summaries "github.com/momokii/echo-notes/internal/repository/meeting-summaries"

	sso_models "github.com/momokii/go-sso-web/pkg/models"
	sso_user "github.com/momokii/go-sso-web/pkg/repository/user"
	sso_utils "github.com/momokii/go-sso-web/pkg/utils"
)

type SummariesGroupHandler struct {
	openaiClient              openai.OpenAI
	dbService                 databases.DBService
	userRepo                  sso_user.UserRepo
	meetingGroupSummariesRepo meeting_group_summaries.MeetingGroupSummaries
	meetingSummariesRepo      meeting_summaries.MeetingSummaries
}

func NewSummariesGroupHandler(
	openaiClient openai.OpenAI,
	dbService databases.DBService,
	userRepo sso_user.UserRepo,
	meetingGroupSummariesRepo meeting_group_summaries.MeetingGroupSummaries,
	meetingSummariesRepo meeting_summaries.MeetingSummaries,
) *SummariesGroupHandler {
	return &SummariesGroupHandler{
		openaiClient:              openaiClient,
		dbService:                 dbService,
		userRepo:                  userRepo,
		meetingGroupSummariesRepo: meetingGroupSummariesRepo,
		meetingSummariesRepo:      meetingSummariesRepo,
	}
}

func (h *SummariesGroupHandler) SummariesGroupReduceUserToken(c *fiber.Ctx) error {

	// get user from session
	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	const FEATURE_COST = utils.MEETING_GROUPING_SUMMARY_AI_COST

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

func (h *SummariesGroupHandler) CreateGroupSummariesDataLLM(c *fiber.Ctx) error {
	var all_summaries_input_message string
	response_llm := new(models.SummarizeGroupLLMResponse)

	session_user := c.Locals("user").(sso_models.UserSession)
	if session_user.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	group_id_input := new(models.SummarizeGroupInput)
	if err := c.BodyParser(group_id_input); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, err.Error())
	}

	// setup llm data prompt and response format
	system_prompt := utils.SYSTEM_PROMPT_GROUPING_SUMMARIES

	messages_llm := []openai.OAMessageReq{
		{
			Role:    "system",
			Content: system_prompt,
		},
	}

	response_format := openai.OACreateResponseFormat(
		"summaries_group_response_format",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"overview":          map[string]interface{}{"type": "string"},
				"meeting_summaries": map[string]interface{}{"type": "string"},
				"next_step":         map[string]interface{}{"type": "string"},
			},
		},
	)

	// check data input for data summary
	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		// find all data need to be summarize from ids input
		summaries_datas, err := h.meetingSummariesRepo.FindByIds(tx, group_id_input.SummariesId, session_user.Id)
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		if len(*summaries_datas) == 0 {
			return fmt.Errorf("Data input not found in your account"), fiber.StatusNotFound
		}

		// for every data, concat to make one big data that contain all comprehensive summary
		var summariesDataInputLLM []models.SummariesGroupLLMInput
		for _, data := range *summaries_datas {
			data_input := models.SummariesGroupLLMInput{
				Name:        data.Name,
				Description: data.Description,
				Summary:     data.ComprehensiveSummaries,
			}

			summariesDataInputLLM = append(summariesDataInputLLM, data_input)
		}

		// convert all summaries data to string
		data_encode, err := json.Marshal(summariesDataInputLLM)
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		all_summaries_input_message = string(data_encode)

		// add all summaries data to llm messages and send req to openai
		messages_llm = append(messages_llm, openai.OAMessageReq{
			Role:    "user",
			Content: all_summaries_input_message,
		})

		response, err := h.openaiClient.OpenAIGetFirstContentDataResp(&messages_llm, true, &response_format, false, nil)
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		if err := json.Unmarshal([]byte(response.Content), &response_llm); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK

	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success create group summaries llm data", fiber.Map{
		"group_summaries_data": response_llm,
	})
}

// TODO: Implement GetGroupSummariesData function (test needed)
func (h *SummariesGroupHandler) GetGroupSummaries(c *fiber.Ctx) error {
	var summariesGroupData []models.MeetingGroupingSummary
	total_data := 0

	user_session := c.Locals("user").(*sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseMessage(c, fiber.StatusUnauthorized, "unauthorized")
	}

	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 10)
	search := c.Query("search", "")

	paginationData := models.PaginationFiltering{
		Page:    page,
		PerPage: perPage,
		Search:  search,
	}

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		summariesGroup, total, err := h.meetingGroupSummariesRepo.Find(tx, paginationData, user_session.Id)
		if err != nil {
			return err, fiber.StatusInternalServerError
		}

		total_data = total
		summariesGroupData = *summariesGroup

		// check if data is empty then return empty array instead of nil
		if summariesGroupData == nil {
			summariesGroupData = []models.MeetingGroupingSummary{}
		}

		return nil, fiber.StatusOK

	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success get group summaries data", fiber.Map{
		"group_summaries": summariesGroupData,
		"pagination": fiber.Map{
			"total_page":   total_data/perPage + 1,
			"total_data":   total_data,
			"current_page": page,
			"per_page":     perPage,
		},
	})
}

// TODO: Implement GetOneGroupSummaries function (test needed)
func (h *SummariesGroupHandler) GetOneGroupSummaries(c *fiber.Ctx) error {
	var meetingGroupSummary models.MeetingGroupingSummary

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id := c.Params("id")
	id_int, err := strconv.Atoi(id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "invalid id")
	}

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		data, err := h.meetingGroupSummariesRepo.FindById(tx, id_int, user_session.Id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Data not found in your account"), fiber.StatusNotFound
			}

			return err, fiber.StatusInternalServerError
		}

		meetingGroupSummary = *data

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseWitData(c, fiber.StatusOK, "success get one group summaries data", fiber.Map{
		"group_summaries": meetingGroupSummary,
	})
}

func (h *SummariesGroupHandler) SaveGroupSummaries(c *fiber.Ctx) error {

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseMessage(c, fiber.StatusUnauthorized, "unauthorized")
	}

	groupSummariesNew := new(models.MeetingGroupingSummaryCreate)
	if err := c.BodyParser(groupSummariesNew); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, err.Error())
	}

	groupSummariesNew.UserId = user_session.Id
	// here for default value for name and description
	groupSummariesNew.Name = "Group Summaries"
	time_now := time.Now().Format("2006-01-02 15:04:05")
	groupSummariesNew.Description = "Group Summaries Description | " + time_now

	// TODO: add validation for groupSummariesNew here

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {
		if err := h.meetingGroupSummariesRepo.Create(tx, *groupSummariesNew); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "success save group summaries data")
}

// TODO: Implement UpdateGroupSummaries function (test needed)
func (h *SummariesGroupHandler) UpdateGroupSummaries(c *fiber.Ctx) error {

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseMessage(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id := c.Params("id")
	id_int, err := strconv.Atoi(id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "invalid id")
	}

	updateGroupSummariesData := new(models.MeetingGroupingSummaryUpdate)
	if err := c.BodyParser(updateGroupSummariesData); err != nil {
		return utils.ResponseError(c, fiber.StatusBadGateway, "failed to parse request body")
	}

	updateGroupSummariesData.UserId = user_session.Id

	// TODO: validate data here

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		// check data exist
		summaries_update, err := h.meetingGroupSummariesRepo.FindById(tx, id_int, user_session.Id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Data not found in your account"), fiber.StatusNotFound
			}

			return err, fiber.StatusInternalServerError
		}

		summaries_update.Name = updateGroupSummariesData.Name
		summaries_update.Description = updateGroupSummariesData.Description

		if err := h.meetingGroupSummariesRepo.Update(tx, *summaries_update); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK

	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "success update group summaries data")
}

// TODO: Implement DeleteGroupSummaries function (test needed)
func (h *SummariesGroupHandler) DeleteGroupSummaries(c *fiber.Ctx) error {

	user_session := c.Locals("user").(sso_models.UserSession)
	if user_session.Id == 0 {
		return utils.ResponseMessage(c, fiber.StatusUnauthorized, "unauthorized")
	}

	deleted_id := c.Params("id")
	deleted_id_int, err := strconv.Atoi(deleted_id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "invalid id")
	}

	if err, code := h.dbService.Transaction(c.Context(), func(tx *sql.Tx) (error, int) {

		// check data
		data_summaries_group, err := h.meetingGroupSummariesRepo.FindById(tx, deleted_id_int, user_session.Id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Data not found in your account"), fiber.StatusNotFound
			}

			return err, fiber.StatusInternalServerError
		}

		// delete data
		if err := h.meetingGroupSummariesRepo.Delete(tx, *data_summaries_group); err != nil {
			return err, fiber.StatusInternalServerError
		}

		return nil, fiber.StatusOK
	}); err != nil {
		return utils.ResponseError(c, code, err.Error())
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "success delete one group summaries data")
}
