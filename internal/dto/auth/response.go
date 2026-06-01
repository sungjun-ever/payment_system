package auth

type Resource struct {
	AccessToken string `json:"access_token"`
}

func NewResource(token string) *Resource {
	return &Resource{token}
}
