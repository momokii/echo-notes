package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/momokii/echo-notes/internal/models"
	"github.com/momokii/echo-notes/pkg/utils"
	"github.com/momokii/go-llmbridge/pkg/openai"
)

type SummariesHandler struct {
	openaiClient openai.OpenAI
}

func NewSummariesHandler(openaiClient openai.OpenAI) *SummariesHandler {
	return &SummariesHandler{
		openaiClient: openaiClient,
	}
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

	type OAReqSpeechToText struct {
		File           *multipart.FileHeader `json:"file" form:"file"`                       // required
		Model          string                `json:"model" form:"model"`                     // required
		Prompt         string                `json:"prompt" form:"prompt"`                   // optional, An optional text to guide the model's style or continue a previous audio segment
		ResponseFormat string                `json:"response_format" form:"response_format"` // default to json, The format of the response. Either json or text, srt, verbose_json, or vtt.
		Temperature    float64               `json:"temperature" form:"temperature"`         // The sampling temperature, between 0 and 1. Higher values like 0.8 will make the output more random, while lower values like 0.2 will make it more focused and deterministic. If set to 0, the model will use log probability to automatically increase the temperature until certain thresholds are hit.
	}

	type OARespSpeechToText struct {
		Text string `json:"text" form:"text"`
	}

	// Get the audio file from the form data
	audioFile := files[0]

	// Set default temperature
	temperature := 0

	// Create the request object
	req := OAReqSpeechToText{
		File:        audioFile,
		Model:       "whisper-1",
		Temperature: float64(temperature),
	}

	// Create a buffer to hold the form data
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the file to the form
	fw, err := w.CreateFormFile("file", audioFile.Filename)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to create form file")
	}

	// Check file format and ensure it's .webm
	fileName := audioFile.Filename

	file, err := audioFile.Open()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to open audio file")
	}
	defer file.Close()

	// Create form file with .webm filename
	fw, err = w.CreateFormFile("file", fileName)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to create form file with webm extension")
	}

	if _, err = io.Copy(fw, file); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to copy audio file")
	}

	// Add other fields to the form
	if fw, err = w.CreateFormField("model"); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to create form field")
	}
	if _, err = fw.Write([]byte(req.Model)); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to write form field")
	}

	if fw, err = w.CreateFormField("temperature"); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to create form field")
	}
	if _, err = fw.Write([]byte(strconv.FormatFloat(req.Temperature, 'f', -1, 64))); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to write form field")
	}

	// Close the writer to finalize the form
	w.Close()

	// Create the HTTP request
	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &b)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to create HTTP request")
	}
	httpReq.Header.Set("Content-Type", w.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to send HTTP request")
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to read response body")
	}

	// Parse the response
	var oaResp OARespSpeechToText
	if err := json.Unmarshal(respBody, &oaResp); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "failed to parse response body")
	}

	// Tambahkan setelah membaca respBody
	log.Printf("Raw response from OpenAI: %s\n", string(respBody))

	return utils.ResponseWitData(c, fiber.StatusOK, "success process chunk", fiber.Map{
		"chunk_number":    chunk_number,
		"translated_text": oaResp.Text,
	})
}

// TODO Check this function when connect to LLM
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

		Part 1: TLDR (Too Long; Didn't Read) Summary
		- Produce a brief, high-level summary (3-5 sentences) that captures the most critical - points, key decisions, and primary action items.
		Ensure it is concise, clear, and written in a formal tone.

		Part 2: Comprehensive Summary
		- Create an in-depth summary that includes:
			- An overview of the meeting agenda.
			- Detailed discussion points and context around the topics covered.
			- Specific decisions made, including any deadlines and assigned responsibilities.
			- Additional relevant insights to provide a full picture of the meeting’s outcomes.
		- Format this summary using clear headings and bullet points where appropriate to enhance readability.

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

	return utils.ResponseWitData(c, fiber.StatusOK, "success", fiber.Map{
		"summaries_data": response_data,
	})
}
