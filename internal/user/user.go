// Package user 提供用户领域示例及其持久化边界。
package user

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// User 表示持久化的用户记录。
type User struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	CreatedAt time.Time `json:"created_at"`
}

// Repository 定义 Service 所需的持久化操作。
type Repository interface {
	Create(context.Context, *User) error
}

// GORMRepository 使用 GORM 持久化用户。
type GORMRepository struct {
	DB *gorm.DB
}

// Create 使用请求上下文插入用户记录。
func (r GORMRepository) Create(ctx context.Context, u *User) error {
	return r.DB.WithContext(ctx).Create(u).Error
}

// Service 协调用户业务操作。
type Service struct {
	Repository Repository
}

// Create 根据指定名称创建并持久化用户。
func (s Service) Create(ctx context.Context, name string) (*User, error) {
	u := &User{Name: name}
	if err := s.Repository.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
