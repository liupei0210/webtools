package page

import "gorm.io/gorm"

func PageTemplate[T interface{}](pageNum, pageSize int, handler func() *gorm.DB) (*Page[T], error) {
	var total int64
	results := make([]T, 0)
	query := handler()
	query.Count(&total)
	if err := query.Limit(pageSize).Offset((pageNum - 1) * pageSize).Find(&results).Error; err != nil {
		return nil, err
	}
	return &Page[T]{
		CurrentSize: len(results),
		TotalSize:   total,
		Content:     &results,
	}, nil
}

type PageReq struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}
type Page[T interface{}] struct {
	Content     *[]T  `json:"content"`
	CurrentSize int   `json:"currentSize"`
	TotalSize   int64 `json:"totalSize"`
}

func GetPageReq(number, size int) PageReq {
	return PageReq{
		PageNum:  number,
		PageSize: size,
	}
}
