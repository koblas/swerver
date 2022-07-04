package minimatch_test

import (
	"testing"

	"github.com/koblas/swerver/pkg/minimatch"
	"github.com/stretchr/testify/assert"
)

func TestBalanceMatchBasic(t *testing.T) {
	r, err := minimatch.BalancedMatch("{", "}", "pre{in{nest}}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 3)
	assert.Equal(t, r.End, 12)
	assert.Equal(t, r.Pre, "pre")
	assert.Equal(t, r.Body, "in{nest}")
	assert.Equal(t, r.Post, "post")
}

func TestBalanceMatchDeep(t *testing.T) {
	r, err := minimatch.BalancedMatch("{", "}", "{{{{{{{{{in}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 8)
	assert.Equal(t, r.End, 11)
	assert.Equal(t, r.Pre, "{{{{{{{{")
	assert.Equal(t, r.Body, "in")
	assert.Equal(t, r.Post, "post")
}

func TestBalanceMatchMismatch1(t *testing.T) {
	r, err := minimatch.BalancedMatch("{", "}", "pre{body{in}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 8)
	assert.Equal(t, r.End, 11)
	assert.Equal(t, r.Pre, "pre{body")
	assert.Equal(t, r.Body, "in")
	assert.Equal(t, r.Post, "post")
}

func TestBalanceMatchMismatch2(t *testing.T) {
	r, err := minimatch.BalancedMatch("{", "}", "pre{in}po}st")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 3)
	assert.Equal(t, r.End, 6)
	assert.Equal(t, r.Pre, "pre")
	assert.Equal(t, r.Body, "in")
	assert.Equal(t, r.Post, "po}st")
}

func TestBalanceMatchMismatch3(t *testing.T) {
	r, err := minimatch.BalancedMatch("{", "}", "pre}{in{nest}}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 4)
	assert.Equal(t, r.End, 13)
	assert.Equal(t, r.Pre, "pre}")
	assert.Equal(t, r.Body, "in{nest}")
	assert.Equal(t, r.Post, "post")
}

func TestBalanceMatchMismatch4(t *testing.T) {
	r, err := minimatch.BalancedMatch("{", "}", "pre{body}between{body2}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 3)
	assert.Equal(t, r.End, 8)
	assert.Equal(t, r.Pre, "pre")
	assert.Equal(t, r.Body, "body")
	assert.Equal(t, r.Post, "between{body2}post")
}

func TestBalanceMatchHtml1(t *testing.T) {
	r, err := minimatch.BalancedMatch("<b>", "</b>", "pre<b>in<b>nest</b></b>post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 3)
	assert.Equal(t, r.End, 19)
	assert.Equal(t, r.Pre, "pre")
	assert.Equal(t, r.Body, "in<b>nest</b>")
	assert.Equal(t, r.Post, "post")
}

func TestBalanceMatchHmlt2(t *testing.T) {
	r, err := minimatch.BalancedMatch("<b>", "</b>", "pre</b><b>in<b>nest</b></b>post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 7)
	assert.Equal(t, r.End, 23)
	assert.Equal(t, r.Pre, "pre</b>")
	assert.Equal(t, r.Body, "in<b>nest</b>")
	assert.Equal(t, r.Post, "post")
}

func TestBalanceMatchParen1(t *testing.T) {
	r, err := minimatch.BalancedMatch("{{", "}}", "pre{{{in}}}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 3)
	assert.Equal(t, r.End, 9)
	assert.Equal(t, r.Pre, "pre")
	assert.Equal(t, r.Body, "{in}")
	assert.Equal(t, r.Post, "post")
}

func TestBalanceMatchParen2(t *testing.T) {
	r, err := minimatch.BalancedMatch("{{{", "}}", "pre{{{in}}}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 3)
	assert.Equal(t, r.End, 8)
	assert.Equal(t, r.Pre, "pre")
	assert.Equal(t, r.Body, "in")
	assert.Equal(t, r.Post, "}post")
}

func TestBalanceMatchParen3(t *testing.T) {
	r, err := minimatch.BalancedMatch("{", "}", "pre{{first}in{second}post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 4)
	assert.Equal(t, r.End, 10)
	assert.Equal(t, r.Pre, "pre{")
	assert.Equal(t, r.Body, "first")
	assert.Equal(t, r.Post, "in{second}post")
}

func TestBalanceMatchParen(t *testing.T) {
	r, err := minimatch.BalancedMatch("<?", "?>", "pre<?>post")

	assert.Nil(t, err, "Error is non-nil")
	assert.Equal(t, r.Start, 3)
	assert.Equal(t, r.End, 4)
	assert.Equal(t, r.Pre, "pre")
	assert.Equal(t, r.Body, "")
	assert.Equal(t, r.Post, "post")
}

/*
t.deepEqual(balanced('<?', '?>', 'pre<?>post'), {
	start: 3,
	end: 4,
	pre: 'pre',
	body: '',
	post: 'post'
});
*/

func TestBalanceMatchError1(t *testing.T) {
	_, err := minimatch.BalancedMatch("{", "}", "nope")

	assert.NotNil(t, err, "Error is non-nil")
}

func TestBalanceMatchError2(t *testing.T) {
	_, err := minimatch.BalancedMatch("{", "}", "{nope")

	assert.NotNil(t, err, "Error is non-nil")
}

func TestBalanceMatchError3(t *testing.T) {
	_, err := minimatch.BalancedMatch("{", "}", "nope}")

	assert.NotNil(t, err, "Error is non-nil")
}
