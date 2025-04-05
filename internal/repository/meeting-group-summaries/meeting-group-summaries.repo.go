package meeting_group_summaries

import (
	"database/sql"
	"fmt"

	"github.com/momokii/echo-notes/internal/models"
)

type MeetingGroupSummaries struct{}

func NewMeetingGroupSummaries() *MeetingGroupSummaries {
	return &MeetingGroupSummaries{}
}

func (m *MeetingGroupSummaries) Find(tx *sql.Tx, pagination models.PaginationFiltering, userId int) (*[]models.MeetingGroupingSummary, int, error) {
	var meetingGroupSummaries []models.MeetingGroupingSummary

	offset := (pagination.Page - 1) * pagination.PerPage
	total := 0

	idxParam := 1
	paramData := []interface{}{}

	total_query := "SELECT COUNT(id) FROM meeting_grouping_summaries WHERE 1=1"
	query := "SELECT id, name, description, user_id, overview, meeting_summaries, next_steps, created_at, updated_at FROM meeting_grouping_summaries WHERE 1=1"

	// check user id and add to query
	if userId < 1 {
		return nil, 0, fmt.Errorf("user id is required")
	}
	add_query := " AND user_id = $" + fmt.Sprint(idxParam)
	query += add_query
	total_query += add_query
	idxParam++
	paramData = append(paramData, userId)

	// check if pagination using search query
	if pagination.Search != "" {
		add_query := " AND (name ILIKE '%' || $" + fmt.Sprint(idxParam) + " || '%' OR description ILIKE '%' || $" + fmt.Sprint(idxParam) + " || '%')"
		query += add_query
		total_query += add_query
		idxParam++
		paramData = append(paramData, pagination.Search)
	}

	// count data
	if err := tx.QueryRow(total_query, paramData...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// query data
	query += " ORDER BY id DESC LIMIT $" + fmt.Sprint(idxParam) + " OFFSET $" + fmt.Sprint(idxParam+1)
	idxParam += 2
	paramData = append(paramData, pagination.PerPage, offset)

	rows, err := tx.Query(query, paramData...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var meetingGroupSummary models.MeetingGroupingSummary

		err := rows.Scan(
			&meetingGroupSummary.Id,
			&meetingGroupSummary.Name,
			&meetingGroupSummary.Description,
			&meetingGroupSummary.UserId,
			&meetingGroupSummary.Overview,
			&meetingGroupSummary.MeetingSummaries,
			&meetingGroupSummary.NextSteps,
			&meetingGroupSummary.CreatedAt,
			&meetingGroupSummary.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		meetingGroupSummaries = append(meetingGroupSummaries, meetingGroupSummary)
	}

	return &meetingGroupSummaries, total, nil
}

func (m *MeetingGroupSummaries) FindById(tx *sql.Tx, id, userId int) (*models.MeetingGroupingSummary, error) {
	var meetingGroupSummary models.MeetingGroupingSummary

	query := "SELECT id, name, description, user_id,  overview, meeting_summaries, next_steps, created_at, updated_at FROM meeting_grouping_summaries WHERE id = $1 AND user_id = $2"

	if err := tx.QueryRow(query, id, userId).Scan(
		&meetingGroupSummary.Id,
		&meetingGroupSummary.Name,
		&meetingGroupSummary.Description,
		&meetingGroupSummary.UserId,
		&meetingGroupSummary.Overview,
		&meetingGroupSummary.MeetingSummaries,
		&meetingGroupSummary.NextSteps,
		&meetingGroupSummary.CreatedAt,
		&meetingGroupSummary.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &meetingGroupSummary, nil
}

func (m *MeetingGroupSummaries) Create(tx *sql.Tx, meetingGroupSummariesData models.MeetingGroupingSummaryCreate) error {

	query := "INSERT INTO meeting_grouping_summaries (name, description, user_id, overview, meeting_summaries, next_steps) VALUES ($1, $2, $3, $4, $5, $6)"

	_, err := tx.Exec(
		query,
		meetingGroupSummariesData.Name,
		meetingGroupSummariesData.Description,
		meetingGroupSummariesData.UserId,
		meetingGroupSummariesData.Overview,
		meetingGroupSummariesData.MeetingSummaries,
		meetingGroupSummariesData.NextSteps,
	)
	if err != nil {
		return err
	}

	return nil
}

func (m *MeetingGroupSummaries) Update(tx *sql.Tx, meetingGroupSummariesData models.MeetingGroupingSummary) error {

	query := "UPDATE meeting_grouping_summaries SET name = $1, description = $2 WHERE id = $3 AND user_id = $4"

	_, err := tx.Exec(
		query,
		meetingGroupSummariesData.Name,
		meetingGroupSummariesData.Description,
		meetingGroupSummariesData.Id,
		meetingGroupSummariesData.UserId,
	)

	if err != nil {
		return err
	}

	return nil
}

func (m *MeetingGroupSummaries) Delete(tx *sql.Tx, meetingGroupSummariesData models.MeetingGroupingSummary) error {

	query := "DELETE FROM meeting_grouping_summaries WHERE id = $1 AND user_id = $2"

	_, err := tx.Exec(query, meetingGroupSummariesData.Id, meetingGroupSummariesData.UserId)
	if err != nil {
		return err
	}

	return nil
}
