package page

import "gorm.io/gorm"

// Template pageNum从1开始
func Template[T interface{}](pageNum, pageSize int, handler func() (*gorm.DB, error)) (page Page[T], err error) {
	var total int64
	results := make([]T, pageSize)
	query, err := handler()
	if err != nil {
		return
	}
	prepareQuery := query.Session(&gorm.Session{PrepareStmt: true})
	prepareQuery.Count(&total)
	if err = prepareQuery.Limit(pageSize).Offset((pageNum - 1) * pageSize).Find(&results).Error; err != nil {
		return
	}
	page = Page[T]{
		CurrentSize: len(results),
		TotalSize:   total,
		Content:     results,
	}
	return
}

type Req struct {
	PageNum  int `json:"page_num"`
	PageSize int `json:"page_size"`
}
type Page[T interface{}] struct {
	Content     []T   `json:"content"`
	CurrentSize int   `json:"current_size"`
	TotalSize   int64 `json:"total_size"`
}

func GetPageReq(number, size int) Req {
	return Req{
		PageNum:  number,
		PageSize: size,
	}
}
