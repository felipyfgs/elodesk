package dto

type MetaResp struct {
	TotalCount  int `ltotal_count`
	CurrentPage int `tcurrent_page`
	TotalPages  int `ltotal_pages`
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
