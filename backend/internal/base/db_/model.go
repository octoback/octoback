package db_

import (
	"gorm.io/gorm"
)

func NewBaseModel(id *int64) BaseModel {
	return BaseModel{Id: id}
}

type IdDao[T BaseModel] interface {
	NewEntity(id int64) IdDao[T]
}

type BaseModel struct {
	Id        *int64 `json:"id" gorm:"PrimaryKey;column:id;comment:主键ID" example:"7"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime:milli;column:created_at;comment:创建时间;" example:"创建时间"`
	UpdatedAt int64  `json:"updatedAt" gorm:"autoUpdateTime:milli;column:updated_at;comment:更新时间;" example:"更新时间"`
}

type LogicDelete struct {
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;column:deleted_at;comment:删除时间" example:"删除时间"`
}
