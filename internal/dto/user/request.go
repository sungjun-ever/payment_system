package user

type CreateRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Name            string `json:"name" binding:"required"`
	Password        string `json:"password" binding:"required"`
	PasswordConfirm string `json:"password_confirm" binding:"required,eqfield=Password"`
}
