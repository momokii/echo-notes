package models

type MeetingSummaries struct {
	Id                     int    `json:"id"`
	Name                   string `json:"name"`
	Description            string `json:"description"`
	UserId                 int    `json:"user_id"`
	SimpleSummaries        string `json:"simple_summaries"`
	ComprehensiveSummaries string `json:"comprehensive_summaries"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

type MeetingSummariesCreate struct {
	Name                   string `json:"name" validate:"required"`
	Description            string `json:"description" validate:"required"`
	UserId                 int    `json:"user_id" validate:"required"`
	SimpleSummaries        string `json:"simple_summaries" validate:"required"`
	ComprehensiveSummaries string `json:"comprehensive_summaries" validate:"required"`
}

type MeetingSummariesUpdate struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	UserId      int    `json:"user_id" validate:"required"`
}
