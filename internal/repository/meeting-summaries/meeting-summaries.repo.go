package meeting_summaries

import (
	"database/sql"
	"fmt"

	"github.com/momokii/echo-notes/internal/models"
)

type MeetingSummaries struct{}

func NewMeetingSummaries() *MeetingSummaries {
	return &MeetingSummaries{}
}

func (m *MeetingSummaries) Find(tx *sql.Tx, pagination models.PaginationFiltering, userId int) (*[]models.MeetingSummaries, int, error) {
	var meetingSummaries []models.MeetingSummaries

	offset := (pagination.Page - 1) * pagination.PerPage
	total := 0

	idxParam := 1
	paramData := []interface{}{}

	total_query := "SELECT COUNT(id) FROM meeting_summaries WHERE 1=1"
	query := "SELECT id, name, description, user_id, simple_summaries, comprehensive_summaries, created_at, updated_at FROM meeting_summaries WHERE 1=1"

	// check user id and add to query
	if userId < 1 {
		return nil, 0, fmt.Errorf("user id is required")
	}
	add_query := " AND user_id = $" + fmt.Sprint(idxParam)
	query += add_query
	total_query += add_query
	idxParam++
	paramData = append(paramData, userId)

	// check if using search query
	if pagination.Search != "" {
		add_query := " AND (name ILIKE '%' || $" + fmt.Sprint(idxParam) + " || '%' OR description ILIKE '%' || $" + fmt.Sprint(idxParam) + " || '%')"
		query += add_query
		total_query += add_query
		idxParam++
		paramData = append(paramData, pagination.Search)
	}

	// count total data
	if err := tx.QueryRow(total_query, paramData...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// add order by status, limit and offset to query for get data
	if pagination.OrderBy == models.ORDER_BY_OLDEST {
		query += " ORDER BY id ASC"
	} else {
		query += " ORDER BY id DESC"
	}
	query += " LIMIT $" + fmt.Sprint(idxParam) + " OFFSET $" + fmt.Sprint(idxParam+1)
	idxParam += 2
	paramData = append(paramData, pagination.PerPage, offset)

	// get data
	rows, err := tx.Query(query, paramData...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var meetingSummary models.MeetingSummaries

		if err := rows.Scan(
			&meetingSummary.Id,
			&meetingSummary.Name,
			&meetingSummary.Description,
			&meetingSummary.UserId,
			&meetingSummary.SimpleSummaries,
			&meetingSummary.ComprehensiveSummaries,
			&meetingSummary.CreatedAt,
			&meetingSummary.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		meetingSummaries = append(meetingSummaries, meetingSummary)
	}

	return &meetingSummaries, total, nil
}

func (m *MeetingSummaries) FindById(tx *sql.Tx, id, userId int) (*models.MeetingSummaries, error) {
	var meetingSummaries models.MeetingSummaries

	query := "SELECT id, name, description, user_id, simple_summaries, comprehensive_summaries, created_at, updated_at FROM meeting_summaries WHERE id = $1 AND user_id = $2"

	if err := tx.QueryRow(query, id, userId).Scan(
		&meetingSummaries.Id,
		&meetingSummaries.Name,
		&meetingSummaries.Description,
		&meetingSummaries.UserId,
		&meetingSummaries.SimpleSummaries,
		&meetingSummaries.ComprehensiveSummaries,
		&meetingSummaries.CreatedAt,
		&meetingSummaries.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &meetingSummaries, nil
}

func (m *MeetingSummaries) Create(tx *sql.Tx, meetingSummariesData models.MeetingSummariesCreate) error {

	query := "INSERT INTO meeting_summaries (name, description, user_id, simple_summaries, comprehensive_summaries) VALUES ($1, $2, $3, $4, $5)"

	_, err := tx.Exec(
		query,
		meetingSummariesData.Name,
		meetingSummariesData.Description,
		meetingSummariesData.UserId,
		meetingSummariesData.SimpleSummaries,
		meetingSummariesData.ComprehensiveSummaries,
	)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Implement the following methods update meeting summaries (testing)
func (m *MeetingSummaries) Update(tx *sql.Tx, meetingSummariesData models.MeetingSummaries) error {

	query := "UPDATE meeting_summaries SET name = $1, description = $2 WHERE id = $3 AND user_id = $4"

	_, err := tx.Exec(
		query,
		meetingSummariesData.Name,
		meetingSummariesData.Description,
		meetingSummariesData.Id,
		meetingSummariesData.UserId,
	)

	if err != nil {
		return err
	}

	return nil
}

// TODO: Implement the following methods delete meeting summaries (testing)
func (m *MeetingSummaries) Delete(tx *sql.Tx, meetingSummariesData models.MeetingSummaries) error {

	query := "DELETE FROM meeting_summaries WHERE id = $1 AND user_id = $2"

	_, err := tx.Exec(query, meetingSummariesData.Id, meetingSummariesData.UserId)
	if err != nil {
		return err
	}

	return nil
}
