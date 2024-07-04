package format

import (
	"encoding/json"
	"io"
	"math"
)

type PaginationSource interface {
	Count() (int64, error)
	Data(limit, p int64, format ObjToMapFunc) ([]map[string]interface{}, error)
}

type Pagination interface {
	Output(w io.Writer) error
	GetRows() interface{}
	GetAllPages() int64
	GetPage() int64
}

type paginationImpl struct {
	Rows     interface{} `json:"rows,omitempty"`
	Total    int64       `json:"total,omitempty"`
	AllPages int64       `json:"allPages,omitempty"`
	Page     int64       `json:"page,omitempty"`
	Limit    int64       `json:"limit,omitempty"`
}

const (
	MaxLimit = 300
)

func (pi *paginationImpl) Output(w io.Writer) error {
	return json.NewEncoder(w).Encode(pi)
}

func (pi *paginationImpl) GetRows() interface{} {
	return pi.Rows
}
func (pi *paginationImpl) GetAllPages() int64 {
	return pi.AllPages
}
func (pi *paginationImpl) GetPage() int64 {
	return pi.Page
}

func NewPagination(
	source PaginationSource,
	limit, page int64,
	format func(i interface{}) map[string]interface{},
) (Pagination, error) {
	total, err := source.Count()

	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}
	if limit < 1 || limit > MaxLimit {
		limit = 100
	}
	totalPage := int64(math.Ceil(float64(total) / float64(limit)))

	if page > totalPage {
		page = totalPage
	} else if page < 1 {
		page = 1
	}
	result, err := source.Data(limit, page, format)
	if err != nil {
		return nil, err
	}
	return &paginationImpl{
		Rows:     result,
		Total:    total,
		AllPages: totalPage,
		Page:     page,
		Limit:    limit,
	}, nil
}
