package middleware

import (
	"errors"
	"gitee.com/zhucheer/orange/app"
)

type Auth struct {
}

func NewAuth() *Auth {
	return &Auth{}
}

// Func implements Middleware interface.
func (w Auth) Func() app.MiddlewareFunc {
	return func(next app.HandlerFunc) app.HandlerFunc {
		return func(c *app.Context) error {

			// 中间件处理逻辑
			if c.Request().Header.Get("auth") == "" {
				c.ResponseWrite([]byte("auth middleware break"))
				return errors.New("auth middleware break")
			}

			return next(c)
		}
	}
}