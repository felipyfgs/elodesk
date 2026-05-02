package dto

type MetaResp struct {
	TotalCount  int `json:"total_count"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}

func NewMetaResp(totalCount, currentPage, perPage int) MetaResp {
	totalPages := 0
	if perPage > 0 {
		totalPages = (totalCount + perPage - 1) / perPage
	}
	return MetaResp{
		TotalCount:  totalCount,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
	}
}

type PaginatedResp[T any] struct {
	Meta    MetaResp `json:"meta"`
	Payload []T      `json:"payload"`
}
