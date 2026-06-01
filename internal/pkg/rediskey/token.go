package rediskey

import "strconv"

const (
	tokenKeyPrefix = "token:"
)

func RefreshToken(userId uint) string {
	return tokenKeyPrefix + "refresh:" + "users:" + strconv.Itoa(int(userId))
}
