package minimatch

import (
	"errors"
	"strings"
)

// npm balanced-match
// MIT License

type BalanceMatchResult struct {
	Start int
	End   int
	Pre   string
	Body  string
	Post  string
}

// BalanceMatch is an implementation of the npm package balance-match.  This the outter
// matching pair of "braced" strings for a given string.  This does not implement regular
// expression matching.
func BalancedMatch(prefix, suffix, str string) (BalanceMatchResult, error) {
	result := balanceMatchRange(prefix, suffix, str)
	if result == nil {
		return BalanceMatchResult{}, errors.New("No match found")
	}

	start := (*result)[0]
	end := (*result)[1]

	br := BalanceMatchResult{
		Start: start,
		End:   end,
		Pre:   str[:start],
	}

	if start+len(prefix) > end {
		br.Body = ""
	} else {
		br.Body = str[start+len(prefix) : end]
	}
	if end+len(suffix) > len(str) {
		br.Post = ""
	} else {
		br.Post = str[end+len(suffix):]
	}

	return br, nil
}

func indexOf(str string, substr string, offset int) int {
	value := strings.Index(str[offset:], substr)
	if value < 0 {
		return value
	}
	return value + offset
}

func balanceMatchRange(a string, b string, str string) *[]int {
	ai := indexOf(str, a, 0)
	bi := indexOf(str, b, ai+1)
	i := ai

	var result *[]int

	if ai >= 0 && bi > 0 {
		begs := make([]int, 0)
		left := len(str)
		right := 0

		for i >= 0 && result == nil {
			if i == ai {
				begs = append(begs, i)
				ai = indexOf(str, a, i+1)
			} else if len(begs) == 1 {
				result = &[]int{begs[0], bi}
				begs = make([]int, 0)
			} else {
				var beg int
				beg, begs = begs[len(begs)-1], begs[:len(begs)-1]

				if beg < left {
					left = beg
					right = bi
				}

				bi = indexOf(str, b, i+1)
			}

			if ai < bi && ai >= 0 {
				i = ai
			} else {
				i = bi
			}
		}

		if len(begs) != 0 {
			result = &[]int{left, right}
		}
	}

	return result
}
