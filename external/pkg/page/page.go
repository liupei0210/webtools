package page

import "gorm.io/gorm"

// Template pageNum从1开始
func Template[T interface{}](req Req, handler func() (*gorm.DB, error)) (page Page[T], err error) {
	var total int64
	results := make([]T, req.PageSize)
	query, err := handler()
	if err != nil {
		return
	}
	prepareQuery := query.Session(&gorm.Session{PrepareStmt: true})
	prepareQuery.Count(&total)
	if err = prepareQuery.Limit(req.PageSize).Offset((req.PageNum - 1) * req.PageSize).Find(&results).Error; err != nil {
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
