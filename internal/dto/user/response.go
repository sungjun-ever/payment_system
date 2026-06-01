package user

import "payment_system/internal/model"

type Resource struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func NewResource(u *model.User) *Resource {
	return &Resource{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
