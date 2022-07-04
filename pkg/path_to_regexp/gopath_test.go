package path_to_regexp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSmokeTest(t *testing.T) {
	keys := []Token{}
	r, err := PathToRegexp("/:foo/:bar", &keys, NewOptions())

	fmt.Printf("%#v\n", keys)

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, 2, len(keys))

	assert.True(t, r.MatchString("/test/path"))
}
