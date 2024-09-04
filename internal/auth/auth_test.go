package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserId(t *testing.T) {
	tokenSuccess, err := BuildJWTString(1)
	require.NoError(t, err)

	gotOne, err := GetUserId(tokenSuccess)
	require.NoError(t, err)
	gotTwo, err := GetUserId(tokenSuccess)
	require.NoError(t, err)
	assert.Equal(t, gotOne, gotTwo)
}
