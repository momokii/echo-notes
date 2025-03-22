package models

const (
	ORDER_BY_NEWEST = "newest"
	ORDER_BY_OLDEST = "oldest"
)

type PaginationFiltering struct {
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Search  string `json:"search"`
	OrderBy string `json:"order_by"`
}
