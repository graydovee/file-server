package server

import (
	"strings"
	"time"
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
