package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_format(t *testing.T) {
	ti, err := time.Parse(time.RFC3339Nano, "2024-10-24T01:02:03.456789+08:00")
	if err != nil {
		t.Error(err)
	}
	str := format(ti)
	assert.Equal(t, "20241024010203456", str)
}
