package server

import (
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const layout = "20060102150405.000"

func GetTimeStamp() string {
	return format(time.Now())
}

func format(t time.Time) string {
	s := t.Format(layout)
	s = strings.ReplaceAll(s, ".", "")
	return s
}

// jsonError 以统一的 {"error": msg} 结构返回错误
func jsonError(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]string{"error": msg})
}
