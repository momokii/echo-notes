package models

type SummariesData struct {
	FullTranslatedText string `json:"full_translated_text" validate:"required"`
}

type SummarizeLLMResponse struct {
	TLDRSummary          string `json:"tldr_summary"`
	ComprehensiveSummary string `json:"comprehensive_summary"`
}
