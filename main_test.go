package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetEvents(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	res, err := c.GetEvents(ctx)

	assert.Nil(t, err, "expecting nil error")
	assert.NotNil(t, res, "expecting non-nil result")

	assert.Equal(t, 0, res.Events[0].EventID, "expecting correct EventID")
	assert.NotEmpty(t, res.Events[0].EventName, "expecting non-empty face_token")

}
