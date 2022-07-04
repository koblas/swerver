package minimatch

// npm package brace-expansion

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

var (
	escSlash  = "\000SLASH" + strconv.Itoa(int(rand.Int31())) + "\000"
	escOpen   = "\000OPEN" + strconv.Itoa(int(rand.Int31())) + "\000"
	escClose  = "\000CLOSE" + strconv.Itoa(int(rand.Int31())) + "\000"
	escComma  = "\000COMMA" + strconv.Itoa(int(rand.Int31())) + "\000"
	escPeriod = "\000PERIOD" + strconv.Itoa(int(rand.Int31())) + "\000"
)

func BraceExpansion(str string) []string {
	result := []string{}
	if len(str) == 0 {
		return result
	}

	// I don't know why Bash 4.3 does this, but it does.
	// Anything starting with {} will have the first two bytes preserved
	// but *only* at the top level, so {},a}b will not expand to anything,
	// but a{},b}c will be expanded to [a}c,abc].
	// One could argue that this is a bug in Bash, but since the goal of
	// this module is to match Bash's rules, we escape a leading {}
	if strings.HasPrefix(str, "{}") {
		str = "\\{\\}" + str[2:]
	}

	for _, item := range expand(escapeBraces(str), true) {
		result = append(result, unescapeBraces(item))
	}

	return result
}

func escapeBraces(str string) string {
	str = strings.Join(strings.Split(str, "\\\\"), escSlash)
	str = strings.Join(strings.Split(str, "\\{"), escOpen)
	str = strings.Join(strings.Split(str, "\\}"), escClose)
	str = strings.Join(strings.Split(str, "\\,"), escComma)
	str = strings.Join(strings.Split(str, "\\."), escPeriod)

	return str
}

func unescapeBraces(str string) string {
	str = strings.Join(strings.Split(str, escSlash), "\\")
	str = strings.Join(strings.Split(str, escOpen), "{")
	str = strings.Join(strings.Split(str, escClose), "}")
	str = strings.Join(strings.Split(str, escComma), ",")
	str = strings.Join(strings.Split(str, escPeriod), ".")

	return str
}

// Basically just str.split(","), but handling cases
// where we have nested braced sections, which should be
// treated as individual members, like {a,{b,c},d}
func parseCommaParts(str string) []string {
	if len(str) == 0 {
		return []string{""}
	}

	m, err := BalancedMatch("{", "}", str)
	if err != nil {
		return strings.Split(str, ",")
	}

	parts := []string{}

	p := strings.Split(m.Pre, ",")
	p[len(p)-1] += "{" + m.Body + "}"
	postParts := parseCommaParts(m.Post)
	if len(m.Post) != 0 {
		var first string
		first, postParts = postParts[0], postParts[1:]

		p[len(p)-1] += first
		p = append(p, postParts...)
	}

	return append(parts, p...)
}

func numeric(str string) int {
	i, err := strconv.Atoi(str)
	if err == nil {
		return i
	}
	return int(str[0])
}

func embrace(str string) string {
	return "{" + str + "}"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func abs(a int) int {
	if a < 0 {
		return 0 - a
	}
	return a
}

func isPadded(el string) bool {
	v, _ := regexp.MatchString("^-?0\\d", el)
	return v
}

func lte(i, y int) bool {
	return i <= y
}

func gte(i, y int) bool {
	return i >= y
}

func expand(str string, isTop bool) []string {
	expansions := []string{}

	m, err := BalancedMatch("{", "}", str)

	if err != nil || strings.HasSuffix(m.Pre, "$") {
		return []string{str}
	}

	isNumericSequence, err := regexp.MatchString("^-?\\d+\\.\\.-?\\d+(?:\\.\\.-?\\d+)?$", m.Body)
	isAlphaSequence, err := regexp.MatchString("^[a-zA-Z]\\.\\.[a-zA-Z](?:\\.\\.-?\\d+)?$", m.Body)
	isSequence := isNumericSequence || isAlphaSequence
	isOptions := strings.Index(m.Body, ",") >= 0

	if !isSequence && !isOptions {
		// {a},b}
		if ok, _ := regexp.MatchString(",.*\\}", m.Post); ok {
			str = m.Pre + "{" + m.Body + escClose + m.Post
			return expand(str, false)
		}
		return []string{str}
	}

	var n []string

	if isSequence {
		n = strings.SplitAfterN(m.Body, "..", 1)
	} else {
		n = parseCommaParts(m.Body)
		if len(n) == 1 {
			// x{{a,b}}y ==> x{a}y x{b}y
			nv := n[0]
			n = []string{}
			for _, item := range expand(nv, false) {
				n = append(n, embrace(item))
			}

			if len(n) == 1 {
				var post []string
				if len(m.Post) != 0 {
					post = expand(m.Post, false)
				} else {
					post = []string{""}
				}

				vals := []string{}
				for _, item := range post {
					vals = append(vals, m.Pre+n[0]+item)
				}

				return vals
			}
		}
	}

	// at this point, n is the parts, and we know it's not a comma set
	// with a single entry.

	// no need to expand pre, since it is guaranteed to be free of brace-sets
	pre := m.Pre
	var post []string
	if len(m.Post) != 0 {
		post = expand(m.Post, false)
	} else {
		post = []string{""}
	}

	N := []string{}

	if isSequence {
		x := numeric(n[0])
		y := numeric(n[1])
		width := min(len(n[0]), len(n[1]))

		var incr int
		if len(n) == 3 {
			incr = abs(numeric(n[2]))
		} else {
			incr = 1
		}

		test := lte
		reverse := y < x
		if reverse {
			incr *= -1
			test = gte
		}

		pad := false
		for _, item := range n {
			pad = pad || isPadded(item)
		}

		for i := x; test(i, y); i += incr {
			var c string
			if isAlphaSequence {
				c = string(i)
				if c == "\\" {
					c = ""
				}
			} else {
				c = strconv.Itoa(i)
				if pad {
					need := width - len(c)
					if need > 0 {
						if i < 0 {
							c = "-" + strings.Repeat("0", need-1) + c
						} else {
							c = strings.Repeat("0", need) + c
						}
					}
				}
			}

			N = append(N, c)
		}
	} else {
		for _, item := range n {
			for _, e := range expand(item, false) {
				N = append(N, e)
			}
		}
	}

	for _, Nitem := range N {
		for _, postItem := range post {
			expansion := pre + Nitem + postItem
			if isTop || isSequence || len(expansion) != 0 {
				expansions = append(expansions, expansion)
			}
		}
	}

	return expansions
}
