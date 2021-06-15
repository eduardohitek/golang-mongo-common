package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_setClientOptions(t *testing.T) {
	client := setClientOptions("mongodb://localhost", "common-test")
	assert.Equal(t, "common-test", *client.AppName)
}
