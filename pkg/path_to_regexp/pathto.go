package path_to_regexp

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	DEFAULT_DELIMITER = "/"
)

var (
	pathRegexp = regexp.MustCompile(strings.Join([]string{
		// Match escaped characters that would otherwise appear in future matches.
		// This allows the user to escape special characters that won't transform.
		//`(\\\\.)`,
		`(\\.)`,
		// Match Express-style parameters and un-named parameters with a prefix
		// and optional suffixes. Matches appear as:
		//
		// ":test(\\d+)?" => ["test", "\d+", undefined, "?"]
		// "(\\d+)"  => [undefined, undefined, "\d+", undefined]
		`|`,
		// `(?:\\:(\\w+)(?:\\(((?:\\\\.|[^\\\\()])+)\\))?|\\(((?:\\\\.|[^\\\\()])+)\\))([+*?])?`,
		`(?:\:(\w+)(?:\(((?:\\.|[^\\()])+)\))?|\(((?:\\.|[^\\()])+)\))([+*?])?`,
	}, ""))

	escapeStringRE = regexp.MustCompile(`([.+*?=^!:${}()[\]|/\\])`)
	escapeGroupRE  = regexp.MustCompile(`([=!:$/()])`)
)

type Options struct {
	// delimiter The default delimiter for segments. (default: '/')
	Delimiter string
	// whitelist List of characters to consider delimiters when parsing. (default: undefined, any character)
	Whitelist string
	// sensitive When true the regexp will be case sensitive. (default: false)
	Sensitive bool
	// strict When true the regexp allows an optional trailing delimiter to match. (default: false)
	Strict bool
	// start When true the regexp will match from the beginning of the string. (default: true)
	Start bool
	// end When true the regexp will match to the end of the string. (default: true)
	End bool
	// endsWith Optional character, or list of characters, to treat as "end" characters.
	EndsWith string
}

func NewOptions() Options {
	return Options{
		End:   true,
		Start: true,
	}
}

type Token struct {
	path      string // "string" version
	Name      string
	Prefix    string
	Delimiter string
	Optional  bool
	Repeat    bool
	Pattern   string
}

type matcherParser struct {
	regexp *regexp.Regexp
	keys   []Token
}

type Result struct {
	keys    []Token
	Results []string
}

type PathMatcher interface {
	MatchString(string) (bool, Result)
}

// PathToRegexp converts a path to a tokenized regular expression
// based on `:NAME` tokens.
func PathToRegexp(path string, options Options) (PathMatcher, error) {
	matcher := matcherParser{}
	err := matcher.tokensToRegExp(parse(path, options), &matcher.keys, options)

	return &matcher, err
}

func (matcher *matcherParser) MatchString(path string) (bool, Result) {
	return true, Result{
		keys:    matcher.keys,
		Results: []string{},
	}
}

func parse(str string, options Options) []Token {
	tokens := []Token{}
	key := 0
	index := 0
	path := ""
	defaultDelimiter := options.Delimiter
	if defaultDelimiter == "" {
		defaultDelimiter = DEFAULT_DELIMITER
	}
	whitelist := options.Whitelist
	pathEscaped := false

	for loc := pathRegexp.FindStringSubmatchIndex(str); loc != nil; loc = pathRegexp.FindStringSubmatchIndex(str[index:]) {
		match := func(idx int) string {
			s := idx * 2
			e := s + 1

			if loc[s] == -1 || loc[e] == -1 {
				return ""
			}
			return str[index+loc[s] : index+loc[e]]
		}

		m := match(0)
		escaped := match(1)
		offset := loc[0]
		path = path + str[index:index+offset]
		prev := ""
		name := match(2)
		capture := match(3)
		group := match(4)
		modifier := match(5)
		index += offset + len(m)

		// Ignore already escaped sequences.
		if escaped != "" {
			path += string(escaped[1])
			pathEscaped = true
			continue
		}

		if !pathEscaped && len(path) != 0 {
			k := len(path) - 1
			c := rune(path[k])

			matches := true
			if len(whitelist) != 0 {
				matches = strings.IndexRune(whitelist, c) > 0
			}

			if matches {
				prev = string(c)
				path = path[0:k]
			}
		}

		// Push the current path onto the tokens.
		if path != "" {
			tokens = append(tokens, Token{path: path})
			path = ""
			pathEscaped = false
		}

		repeat := false
		optional := false
		if modifier == "+" || modifier == "*" {
			repeat = true
		}
		if modifier == "?" || modifier == "*" {
			optional = true
		}
		pattern := group
		if len(capture) != 0 {
			pattern = capture
		}
		delimiter := DEFAULT_DELIMITER
		if len(prev) != 0 {
			delimiter = prev
		}

		tokName := name
		if len(name) == 0 {
			tokName = strconv.Itoa(key)
			key++
		}
		if len(pattern) != 0 {
			pattern = escapeGroup(pattern)
		} else {
			if delimiter == defaultDelimiter {
				pattern = "[^" + escapeString(delimiter) + "]+?"
			} else {
				pattern = "[^" + escapeString(delimiter+defaultDelimiter) + "]+?"
			}
		}

		tokens = append(tokens, Token{
			Name:      tokName,
			Prefix:    prev,
			Delimiter: delimiter,
			Optional:  optional,
			Repeat:    repeat,
			Pattern:   pattern,
		})
	}

	// Push any remaining characters.
	if len(path) != 0 && index < len(str) {
		tokens = append(tokens, Token{path: str[index:]})
	}

	return tokens
}

func (matcher *matcherParser) tokensToRegExp(tokens []Token, keys *[]Token, options Options) error {
	strict := options.Strict
	start := options.Start
	end := options.End
	delimiter := options.Delimiter
	if delimiter == "" {
		delimiter = DEFAULT_DELIMITER
	}
	endsWith := "$"
	if options.EndsWith != "" {
		endsWith = escapeString(options.EndsWith) + "|$"
	}

	route := ""
	if start {
		route = "^"
	}

	// Iterate over the tokens and create our regexp string.
	for _, token := range tokens {
		if token.path != "" {
			route += escapeString(token.path)
		} else {
			var capture string
			if token.Repeat {
				capture = "(?:" + token.Pattern + ")(?:" + escapeString(token.Delimiter) + "(?:" + token.Pattern + "))*"
			} else {
				capture = token.Pattern
			}

			if keys != nil {
				*keys = append(*keys, token)
			}

			if token.Optional {
				if token.Prefix == "" {
					route += "(" + capture + ")?"
				} else {
					route += "(?:" + escapeString(token.Prefix) + "(" + capture + "))?"
				}
			} else {
				route += escapeString(token.Prefix) + "(" + capture + ")"
			}
		}

	}

	if end {
		if !strict {
			route += "(?:" + escapeString(delimiter) + ")?"
		}

		if endsWith == "$" {
			route += "$"
		} else {
			route += "(?=" + endsWith + ")"
		}
	} else {
		isEndDelimited := true

		if len(tokens) > 0 {
			endToken := tokens[len(tokens)-1]

			if endToken.path != "" {
				isEndDelimited = string(endToken.path[len(endToken.path)-1]) == delimiter
			} else {
				isEndDelimited = false
			}
		}

		if !strict {
			route += "(?:" + escapeString(delimiter) + "(?=" + endsWith + "))?"
		}

		if !isEndDelimited {
			route += "(?=" + escapeString(delimiter) + "|" + endsWith + ")"
		}
	}

	if !options.Sensitive {
		route = "(?i)" + route
	}

	var err error
	matcher.regexp, err = regexp.Compile(route)

	return err
}

func escapeString(str string) string {
	return escapeStringRE.ReplaceAllString(str, `\$1`)
}

func escapeGroup(str string) string {
	return escapeGroupRE.ReplaceAllString(str, `\$1`)
}

// TODO: This needs to work
func Compile(path string) func(map[string]string) string {
	toPath := func(params map[string]string) string {
		return path
	}

	return toPath
}
