package page

import (
	"errors"
	"gorm.io/gorm"
)

// Template 优化分页查询模板函数
func Template[T interface{}](req Req, handler func() (*gorm.DB, error)) (page Page[T], err error) {
	if err = req.validate(); err != nil {
		return
	}

	var total int64
	results := make([]T, 0, req.PageSize) // 优化: 初始容量设为0,避免内存浪费

	query, err := handler()
	if err != nil {
		return
	}

	countQuery := query.Session(&gorm.Session{NewDB: true}).Limit(-1).Offset(-1)

	// 将克隆后的查询作为子查询，统计总数
	if err = countQuery.Table("(?) as subquery", countQuery).Count(&total).Error; err != nil {
		return
	}
	// 应用分页参数到原查询，获取结果
	if total > 0 {
		if err = query.Limit(req.PageSize).Offset((req.PageNum - 1) * req.PageSize).Find(&results).Error; err != nil {
			return
		}
	}
	page = Page[T]{
		Content:     results,
		CurrentSize: len(results),
		TotalSize:   total,
		TotalPages:  (total + int64(req.PageSize) - 1) / int64(req.PageSize), // 新增总页数
	}
	return
}

type Req struct {
	PageNum  int `json:"page_num" validate:"required|min:1"`
	PageSize int `json:"page_size" validate:"required|min:1|max:100000"`
}

func (r Req) validate() error {
	if r.PageNum < 1 {
		return errors.New("页码必须大于0")
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		return errors.New("每页条数必须在1-100之间")
	}
	return nil
}

type Page[T interface{}] struct {
	Content     []T   `json:"content"`
	CurrentSize int   `json:"current_size"`
	TotalSize   int64 `json:"total_size"`
	TotalPages  int64 `json:"total_pages"` // 新增总页数字段
}

func GetPageReq(number, size int) Req {
	return Req{
		PageNum:  number,
		PageSize: size,
	}
}
