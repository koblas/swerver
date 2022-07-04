package minimatch_test

import (
	"testing"

	"github.com/koblas/swerver/pkg/minimatch"
	"github.com/stretchr/testify/assert"
)

func TestBraceExpansion(t *testing.T) {
	r := minimatch.BraceExpansion("file-{a,b,c}.jpg")

	assert.ElementsMatch(t, r, []string{
		"file-a.jpg", "file-b.jpg", "file-c.jpg",
	})
}
