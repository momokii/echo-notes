package models

// recording summaries
type SummariesData struct {
	FullTranslatedText string `json:"full_translated_text" validate:"required"`
}

type SummarizeLLMResponse struct {
	TLDRSummary          string `json:"tldr_summary"`
	ComprehensiveSummary string `json:"comprehensive_summary"`
}

// grouping summaries
type SummarizeGroupInput struct {
	SummariesId []int `json:"summaries_id" validate:"required"`
}

type SummariesGroupLLMInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Summary     string `json:"summary"`
}

type SummarizeGroupLLMResponse struct {
	Overview         string `json:"overview"`
	MeetingSummaries string `json:"meeting_summaries"`
	NextSteps        string `json:"next_steps"`
}
