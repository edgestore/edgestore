package model

// Pagination is passed as a parameter to limit the total of rows.
type Pagination struct {
	Limit  int
	Offset int
}

func NewPagination(perPage, page int) *Pagination {
	return &Pagination{
		Limit:  perPage,
		Offset: page * perPage,
	}
}
