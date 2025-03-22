package models

type MeetingGroupingSummary struct {
	Id                int    `json:"id" validate:"required"`
	Name              string `json:"name" validate:"required"`
	Description       string `json:"description" validate:"required"`
	UserId            int    `json:"user_id" validate:"required"`
	GroupingSummaries string `json:"grouping_summaries" validate:"required"`
	CreatedAt         string `json:"created_at" validate:"required"`
	UpdatedAt         string `json:"updated_at" validate:"required"`
}

type MeetingGroupingSummaryCreate struct {
	Name              string `json:"name" validate:"required"`
	Description       string `json:"description" validate:"required"`
	UserId            int    `json:"user_id" validate:"required"`
	GroupingSummaries string `json:"grouping_summaries" validate:"required"`
}
