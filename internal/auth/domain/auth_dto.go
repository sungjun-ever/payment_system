package domain

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type Resource struct {
	AccessToken string `json:"access_token"`
}

func NewResource(token string) *Resource {
	return &Resource{token}
}
