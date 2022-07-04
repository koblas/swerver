package minimatch

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
)

type Options struct {
	/**
	 * Debug to stderr
	 */
	Debug bool

	/**
	 * NoBrace-  Do not expand {a,b} and {1..3} brace sets
	 */
	NoBrace bool

	/**
	 * Disable `**` matching against multiple folder names
	 */
	NoGlobStar bool

	///
	// Allow patterns to match filenames starting with a period, even if the pattern does not explicitly have a period in that spot.
	//
	// Note that by default, a/**/b will not match a/.d/b, unless dot is set.
	//
	Dot bool

	/**
	 * Disable "extglob" style patterns like +(a|b).
	 */
	NoExt bool

	/**
	 * Performace a case-insensitive match.
	 */
	NoCase bool

	/**
	* When a match is not found by minimatch.match, return a list containing the pattern itself if this option is set. When not set, an empty list is returned if there are no matches.
	 */
	NoNull bool

	/**
	 *If set, then patterns without slashes will be matched against the basename of the path if it contains slashes. For example, a?b would match the path /xyz/123/acb, but not /xyz/acb/123.
	 */
	MatchBase bool

	/**
	* Suppress the behavior of treating # at the start of a pattern as a comment.
	 */
	NoComment bool

	/**
	 * Suppress the behavior of treating a leading ! character as negation.
	 */
	NoNegate bool

	/**
	 * Returns from negate expressions the same as if they were not negated. (Ie, true on a hit, false on a miss.)
	 */
	FlipNegate bool
}

type Minimatch interface {
	Match(path string, partial bool) bool
	MakeRe() (*regexp.Regexp, error)
}

var (
	emptyRegexp   = regexp.MustCompile(`^$`)
	braceShortcut = regexp.MustCompile(`\{.*\}`)
	slashSplit    = regexp.MustCompile(`/+`)

	// any single thing other than /
	// don't need to escape / when using new RegExp()
	qmark = "[^/]"

	// * => any number of characters
	star = qmark + "*?"

	// ** when dots are allowed.  Anything goes, except .. and .
	// not (^ or / followed by one or two dots followed by $ or /),
	// followed by anything, any number of times.
	twoStarDot = `(?:(?!(?:\/|^)(?:\.{1,2})($|\/)).)*?`

	// not a ^ or / followed by a dot,
	// followed by anything, any number of times.
	twoStarNoDot = `(?:(?!(?:\/|^)\.).)*?`

	// characters that need to be escaped in RegExp.
	reSpecials = "().*{}+?[]^$\\!"

	plTypes = map[string]struct{ open, close string }{
		"!": {open: "(?:(?!(?:", close: "))[^/]*?)"},
		"?": {open: "(?:", close: ")?"},
		"+": {open: "(?:", close: ")+"},
		"*": {open: "(?:", close: ")*"},
		"@": {open: "(?:", close: ")"},
	}

	// Just a value
	GLOBSTAR = regexp.MustCompile("GLOBSTAR")
)

/**
* MatchString  - a strings against the pattern and options
 */
func MatchString(path string, pattern string, options Options) (bool, error) {
	mm, err := NewMinimatch(pattern, options)

	if err != nil {
		return false, err
	}

	return mm.Match(path, false), nil
}

/**
* Match - match a list of strings against the pattern and options
 */
func Match(list []string, pattern string, options Options) []string {
	mm, err := NewMinimatch(pattern, options)

	if err != nil {
		panic(err)
	}

	result := []string{}
	for _, item := range list {
		if mm.Match(item, false) {
			result = append(result, item)
		}
	}

	if options.NoNull && len(result) == 0 {
		return []string{pattern}
	}
	return result
}

type matcher struct {
	/*
		set A 2-dimensional array of regexp or string expressions. Each row in the array corresponds to a brace-expanded pattern. Each item in the row corresponds to a single path-part. For example, the pattern {a,b/c}/d would expand to a set of patterns like:

		[ [ a, d ]
		, [ b, c, d ] ]
		If a portion of the pattern doesn't have any "magic" in it (that is, it's something like "foo" rather than fo*o?), then it will be left as a string rather than converted to a regular expression.
	*/

	/**
	 * regexp Created by the makeRe method. A single regular expression expressing the entire pattern. This is useful in cases where you
	 * wish to use the pattern somewhat like fnmatch(3) with FNM_PATH enabled.
	 */
	regexp *regexp.Regexp

	/**
	 * Negate True if the pattern is negated
	 */
	Negate bool

	/**
	 * Empty True if the pattern is ""
	 */
	Empty bool

	/**
	 * Comment True if the pattern is a comment.
	 */
	Comment bool

	// The input pattern
	pattern string
	// The input options
	options Options

	// is the experssion negated
	negate bool

	// the set of regexps to use
	set [][]*regexp.Regexp

	log *log.Logger
}

func NewMinimatch(pattern string, options Options) (Minimatch, error) {
	pattern = strings.TrimSpace(pattern)

	// windows support: need to use /, not \
	if runtime.GOOS == "windows" {
		pattern = strings.Join(strings.Split(pattern, string(os.PathSeparator)), "/")
	}

	m := &matcher{pattern: pattern, options: options}

	if options.Debug {
		m.log = log.New(os.Stderr, "minimatch:", 0)
	} else {
		m.log = log.New(ioutil.Discard, "", 0)
	}

	if err := m.make(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *matcher) MakeRe() (*regexp.Regexp, error) {
	if err := m.make(); err != nil {
		return nil, err
	}
	return m.regexp, nil
}

func (m *matcher) make() error {
	if m.regexp != nil {
		return nil
	}

	// empty patterns and comments match nothing.
	if !m.options.NoComment && m.pattern[0] == '#' {
		m.Comment = true
		m.regexp = emptyRegexp
		return nil
	}
	if len(m.pattern) == 0 {
		m.Empty = true
		m.regexp = emptyRegexp
		return nil
	}

	// step 1: figure out negation, etc.
	m.parseNegate()

	// step 2: expand braces
	globSet := m.braceExpand()

	// step 3: now we have a set, so turn each one into a series of path-portion
	// matching patterns.
	// These will be regexps, except in the case of "**", which is
	// set to the GLOBSTAR object for globstar behavior,
	// and will not contain any / characters
	globParts := [][]string{}
	for _, s := range globSet {
		globParts = append(globParts, slashSplit.Split(s, -1))
	}

	// glob --> regexps
	m.set = [][]*regexp.Regexp{}
	for _, s := range globParts {
		group := []*regexp.Regexp{}
		allGood := true
		for _, item := range s {
			val, _, _, _ := m.parse(item, false)
			if val != nil {
				// filter out everything that didn't compile properly.
				group = append(group, val)
			} else {
				allGood = false
			}
		}
		if allGood && len(group) != 0 {
			m.set = append(m.set, group)
		}
	}

	return nil
}

func (m *matcher) parseNegate() {
	if m.options.NoNegate {
		return
	}

	pattern := m.pattern
	idx := 0
	for ; idx < len(pattern) && pattern[idx] == '!'; idx++ {
		m.negate = !m.negate
	}

	// Don't copy unless needed
	if idx != 0 {
		m.pattern = m.pattern[idx:len(m.pattern)]
	}
}

/**
 * Brace expansion:
 * a{b,c}d -> abd acd
 * a{b,}c -> abc ac
 * a{0..3}d -> a0d a1d a2d a3d
 * a{b,c{d,e}f}g -> abg acdfg acefg
 * a{b,c}d{e,f}g -> abdeg acdeg abdeg abdfg
 *
 * Invalid sets are not expanded.
 * a{2..}b -> a{2..}b
 * a{b}c -> a{b}c
 */
func (m *matcher) braceExpand() []string {
	if m.options.NoBrace || braceShortcut.MatchString(m.pattern) {
		return []string{m.pattern}
	}

	return BraceExpansion(m.pattern)
}

// parse a component of the expanded set.
// At this point, no pattern may contain "/" in it
// so we're going to return a 2d array, where each entry is the full
// pattern, split on '/', and then turned into a regular expression.
// A regexp is made at the end which joins each array with an
// escaped /, and another full one which joins each regexp with |.
//
// Following the lead of Bash 4.1, note that "**" only has special meaning
// when it is the *only* thing in a path portion.  Otherwise, any series
// of * is equivalent to a single *.  Globstar behavior is enabled by
// default, and can be disabled by setting options.noglobstar.
func (m *matcher) parse(pattern string, isSub bool) (*regexp.Regexp, string, bool, error) {
	if len(pattern) > 64*1024 {
		return nil, "", false, errors.New("pattern is too long")
	}

	if !m.options.NoGlobStar && pattern == "**" {
		return GLOBSTAR, "", false, nil
	}
	if pattern == "" {
		return emptyRegexp, "", false, nil
	}

	type PItem struct {
		kind    string // type
		start   int
		reStart int
		reEnd   int
		open    string
		close   string
	}

	re := ""
	hasMagic := m.options.NoCase
	escaping := false
	// ? => one single character
	patternListStack := []PItem{}
	negativeLists := []PItem{}
	stateChar := ""
	inClass := false
	reClassStart := -1
	classStart := -1

	// . and .. never match anything that doesn't start with .,
	// even when options.dot is set.
	patternStart := ""
	if pattern[0] == '.' {
		patternStart = ""
		//
		// FIXME PCRE expressions
		//	} else if m.options.Dot {
		//		patternStart = "(?!(?:^|\\/)\\.{1,2}(?:$|\\/))"
		// } else {
		//    patternStart = "(?!\\.)"
	}

	clearStateChar := func() {
		// we had some state-tracking character
		// that wasn't consumed by this pass.
		switch stateChar {
		case "":
			// State is already cleared
			return
		case "*":
			re += star
			hasMagic = true
		case "?":
			re += qmark
			hasMagic = true
		default:
			re += "\\" + stateChar
		}

		m.log.Printf("clearStateChar %s %#v\n", stateChar, re)
		stateChar = ""
	}

	for i, c := range pattern {
		m.log.Printf("%s\t%d %s %c\n", pattern, i, re, c)

		// skip over any that are escaped.
		if escaping && strings.ContainsRune(reSpecials, c) {
			re += "\\" + string(c)
			escaping = false
			continue
		}

		switch c {
		case '/':
			// completely not allowed, even escaped.
			// Should already be path-split by now.
			return nil, "", false, errors.New("unexpected /")

		case '\\':
			clearStateChar()
			escaping = true
			continue

		// the various stateChar values
		// for the "extglob" stuff.
		case '?', '*', '+', '@', '!':
			m.log.Printf("%s\t%d %s \"%c\" <-- stateChar", pattern, i, re, c)

			// all of those are literals inside a class, except that
			// the glob [!a] means [^a] in regexp
			if inClass {
				m.log.Println("  in class")
				if c == '!' && i == classStart+1 {
					c = '^'
				}
				re += string(c)
				continue
			}

			// if we already have a stateChar, then it means
			// that there was something like ** or +? in there.
			// Handle the stateChar, then proceed with this one.
			m.log.Printf("call clearStateChar %#v\n", stateChar)
			clearStateChar()
			stateChar = string(c)
			// if extglob is disabled, then +(asdf|foo) isn't a thing.
			// just clear the statechar *now*, rather than even diving into
			// the patternList stuff.
			if m.options.NoExt {
				clearStateChar()
			}
			continue

		case '(':
			if inClass {
				re += "("
				continue
			}

			if stateChar == "" {
				re += "\\("
				continue
			}

			patternListStack = append(patternListStack, PItem{
				kind:    stateChar,
				start:   i - 1,
				reStart: len(re),
				open:    plTypes[stateChar].open,
				close:   plTypes[stateChar].close,
			})
			// negation is (?:(?!js)[^/]*)
			if stateChar == "!" {
				re += "(?:(?!(?:"
			} else {
				re += "(?:"
			}
			m.log.Printf("plType %v %v\n", stateChar, re)
			stateChar = ""
			continue

		case ')':
			if inClass || len(patternListStack) != 0 {
				re += "\\)"
				continue
			}

			clearStateChar()
			hasMagic = true
			var pl PItem
			pl, patternListStack = patternListStack[len(patternListStack)-1], patternListStack[:len(patternListStack)-1]
			// negation is (?:(?!js)[^/]*)
			// The others are (?:<pattern>)<type>
			re += pl.close
			if pl.kind == "!" {
				negativeLists = append(negativeLists, pl)
			}
			pl.reEnd = len(re)
			continue

		case '|':
			if inClass || len(patternListStack) != 0 || escaping {
				re += "\\|"
				escaping = false
				continue
			}

			clearStateChar()
			re += "|"
			continue

		// these are mostly the same in regexp and glob
		case '[':
			// swallow any state-tracking char before the [
			clearStateChar()

			if inClass {
				re += "\\" + string(c)
				continue
			}

			inClass = true
			classStart = i
			reClassStart = len(re)
			re += string(c)
			continue

		case ']':
			//  a right bracket shall lose its special
			//  meaning and represent itself in
			//  a bracket expression if it occurs
			//  first in the list.  -- POSIX.2 2.8.3.2
			if i == classStart+1 || !inClass {
				re += "\\" + string(c)
				escaping = false
				continue
			}

			// handle the case where we left a class open.
			// "[z-a]" is valid, equivalent to "\[z-a\]"
			if inClass {
				// split where the last [ was, make sure we don't have
				// an invalid re. if so, re-walk the contents of the
				// would-be class to re-translate any characters that
				// were passed through as-is
				// TODO: It would probably be faster to determine this
				// without a try/catch and a new RegExp, but it's tricky
				// to do safely.  For now, this is safe and works.
				cs := pattern[classStart+1 : i]

				_, err := regexp.Compile("[" + cs + "]")
				if err != nil {
					// not a valid class!
					_, sp, spMagic, _ := m.parse(cs, true)
					re = re[:reClassStart] + "\\[" + sp + "\\]"
					hasMagic = hasMagic || spMagic
					inClass = false
					continue
				}
			}

			// finish up the class.
			hasMagic = true
			inClass = false
			re += string(c)
			continue

		default:
			// swallow any state char that wasn't consumed
			clearStateChar()

			if escaping {
				// no need
				escaping = false
			} else if strings.ContainsRune(reSpecials, c) && !(c == '^' && inClass) {
				re += "\\"
			}

			re += string(c)
		}
	}

	// handle the case where we left a class open.
	// "[abc" is valid, equivalent to "\[abc"
	if inClass {
		// split where the last [ was, and escape it
		// this is a huge pita.  We now have to re-walk
		// the contents of the would-be class to re-translate
		// any characters that were passed through as-is
		cs := pattern[classStart+1:]
		_, sp, spMagic, _ := m.parse(cs, true)
		re = re[:reClassStart] + "\\[" + sp
		hasMagic = hasMagic || spMagic
	}

	// handle the case where we had a +( thing at the *end*
	// of the pattern.
	// each pattern list stack adds 3 chars, and we need to go through
	// and escape any | chars that were passed through as-is for the regexp.
	// Go through and escape them, taking care not to double-escape any
	// | chars that were already escaped.
	tailRE := regexp.MustCompile(`((?:\\{2}){0,64})(\\?)\|`)

	for _, pl := range patternListStack {
		var tail = re[pl.reStart+len(pl.open):]
		m.log.Println("setting tail", re, pl)
		// maybe some even number of \, then maybe 1 \, followed by a |
		for offset, loc := 0, tailRE.FindStringSubmatchIndex(tail); loc != nil; loc = tailRE.FindStringSubmatchIndex((tail[offset:])) {
			pre := tail[:offset+loc[0]]
			post := tail[offset+loc[1]:]
			p1 := tail[offset+loc[2] : offset+loc[3]]
			p2 := tail[offset+loc[4] : offset+loc[5]]
			if p2 == "" {
				// the | isn't already escaped, so escape it.
				p2 = "\\"
			}
			// need to escape all those slashes *again*, without escaping the
			// one that we need for escaping the | character.  As it works out,
			// escaping an even number of slashes can be done by simply repeating
			// it exactly after itself.  That's why this trick works.
			//
			// I am sorry that you have to see this.
			repl := p1 + p1 + p2 + "|"
			tail = pre + repl + post
			offset = offset + len(repl)
		}

		m.log.Printf("tail=%v\n   %v %v %v\n", tail, tail, pl, re)
		var t string
		switch pl.kind {
		case "*":
			t = star
		case "?":
			t = qmark
		default:
			t = "\\" + pl.kind
		}

		hasMagic = true
		re = re[:pl.reStart] + t + "\\(" + tail
	}

	// handle trailing things that only matter at the very end.
	clearStateChar()
	if escaping {
		// trailing \\
		re += "\\\\"
	}

	// only need to apply the nodot start if the re starts with
	// something that could conceivably capture a dot
	addPatternStart := false
	switch re[0] {
	case '.', '[', '(':
		addPatternStart = true
	}

	// Hack to work around lack of negative lookbehind in JS
	// A pattern like: *.!(x).!(y|z) needs to ensure that a name
	// like 'a.xyz.yz' doesn't match.  So, the first negative
	// lookahead, has to look ALL the way ahead, to the end of
	// the pattern.
	for n := len(negativeLists) - 1; n > -1; n-- {
		nl := negativeLists[n]

		nlBefore := re[:nl.reStart]
		nlFirst := re[nl.reStart : nl.reEnd-8]
		nlLast := re[nl.reEnd-8 : nl.reEnd]
		nlAfter := re[nl.reEnd:]

		nlLast += nlAfter

		// Handle nested stuff like *(*.js|!(*.json)), where open parens
		// mean that we should *not* include the ) in the bit that is considered
		// "after" the negated section.
		openParensBefore := strings.Count(nlBefore, "(") - 1
		cleanAfter := nlAfter
		cleanAfterRe := regexp.MustCompile(`\)[+*?]?`)
		for i := 0; i < openParensBefore; i++ {
			m := cleanAfterRe.FindStringIndex(cleanAfter)
			if m != nil {
				cleanAfter = cleanAfter[:m[0]] + cleanAfter[:m[1]]
			}
		}
		nlAfter = cleanAfter

		dollar := ""
		if nlAfter == "" && !isSub {
			dollar = "$"
		}
		var newRe = nlBefore + nlFirst + nlAfter + dollar + nlLast
		re = newRe
	}

	// if the re is not "" at this point, then we need to make sure
	// it doesn't match against an empty path part.
	// Otherwise a/* will match a/, which it should not.
	//
	// FIXME PCRE Syntax not supported by golang
	// if re != "" && hasMagic {
	//	re = "(?=.)" + re
	// }

	if addPatternStart {
		re = patternStart + re
	}

	// parsing just a piece of a larger pattern.
	if isSub {
		return nil, re, hasMagic, nil
	}

	// skip the regexp for non-magical patterns
	// unescape anything in it, though, so that it'll be
	// an exact match against a file etc.
	/*
		if !hasMagic {
			return globUnescape(pattern)
		}
	*/

	var regExp *regexp.Regexp
	var err error

	if m.options.NoCase {
		regExp, err = regexp.Compile("(?i)^" + re + "$")
	} else {
		regExp, err = regexp.Compile("^" + re + "$")
	}

	if err != nil {
		// If it was an invalid regular expression, then it can't match
		// anything.  This trick looks for a character after the end of
		// the string, which is of course impossible, except in multi-line
		// mode, but it's not a /m regex.
		regExp = regexp.MustCompile("$.")
	}

	//regExp._glob = pattern
	//regExp._src = re

	return regExp, "", false, nil
}

func (m *matcher) Match(f string, partial bool) bool {
	m.log.Println("match", f, m.pattern)
	// short-circuit in the case of busted things.
	// comments, etc.
	if m.Comment {
		return false
	}
	if m.Empty {
		return f == ""
	}

	if f == "/" && partial {
		return true
	}

	// windows: need to use /, not \
	if runtime.GOOS == "windows" {
		f = strings.Join(strings.Split(f, "\\"), "/")
	}

	// treat the test path as a set of pathparts.
	fparts := slashSplit.Split(f, -1)
	m.log.Printf("%#v split %#v\n", m.pattern, fparts)

	// just ONE of the pattern sets in this.set needs to match
	// in order for it to be valid.  If negating, then just one
	// match means that we have failed.
	// Either way, return on the first hit.

	m.log.Println(m.pattern, "set", m.set)

	// Find the basename of the path by looking for the last non-empty segment
	filename := ""
	for i := len(fparts) - 1; filename == "" && i >= 0; i-- {
		filename = fparts[i]
	}

	for _, pattern := range m.set {
		file := fparts
		if m.options.MatchBase && len(pattern) == 1 {
			file = []string{filename}
		}
		var hit = m.matchOne(file, pattern, partial)
		if hit {
			if m.options.FlipNegate {
				return true
			}
			return !m.negate
		}
	}

	// didn't get any hits.  this is success if it's a negative
	// pattern, failure otherwise.
	if m.options.FlipNegate {
		return false
	}
	return m.negate
}

func (m *matcher) matchOne(file []string, pattern []*regexp.Regexp, partial bool) bool {
	m.log.Println("matchOne", file, pattern)

	m.log.Println("matchOne", len(file), len(pattern))

	fi := 0
	pi := 0
	fl := len(file)
	pl := len(pattern)

	for ; fi < fl && pi < pl; fi, pi = fi+1, pi+1 {
		m.log.Println("matchOne loop")
		var p = pattern[pi]
		var f = file[fi]

		m.log.Printf("%v %v %#v\n", pattern, p, f)

		if p == GLOBSTAR {
			m.log.Println("GLOBSTAR", pattern, p, f)

			// "**"
			// a/**/b/**/c would match the following:
			// a/b/x/y/z/c
			// a/x/y/z/b/c
			// a/b/x/b/x/c
			// a/b/c
			// To do this, take the rest of the pattern after
			// the **, and see if it would match the file remainder.
			// If so, return success.
			// If not, the ** "swallows" a segment, and try again.
			// This is recursively awful.
			//
			// a/**/b/**/c matching a/b/x/y/z/c
			// - a matches a
			// - doublestar
			//   - matchOne(b/x/y/z/c, b/**/c)
			//     - b matches b
			//     - doublestar
			//       - matchOne(x/y/z/c, c) -> no
			//       - matchOne(y/z/c, c) -> no
			//       - matchOne(z/c, c) -> no
			//       - matchOne(c, c) yes, hit
			var fr = fi
			var pr = pi + 1
			if pr == pl {
				m.log.Println("** at the end")
				// a ** at the end will just swallow the rest.
				// We have found a match.
				// however, it will not swallow /.x, unless
				// options.dot is set.
				// . and .. are *never* matched by **, for explosively
				// exponential reasons.
				for _, part := range file[fi:] {
					if part == "." || part == ".." || (!m.options.Dot && len(part) != 0 && part[0] == '.') {
						return false
					}
				}
				return true
			}

			// ok, let's see if we can swallow whatever we can.
			for fr < fl {
				swallowee := file[fr]

				m.log.Println("\nglobstar while", file, fr, pattern, pr, swallowee)

				// XXX remove this slice.  Just pass the start index.
				if m.matchOne(file[fr:], pattern[pr:], partial) {
					m.log.Println("globstar found match!", fr, fl, swallowee)
					// found a match.
					return true
				} else {
					// can't swallow "." or ".." ever.
					// can only swallow ".foo" when explicitly asked.
					if swallowee == "." || swallowee == ".." || (!m.options.Dot && swallowee[0] == '.') {
						m.log.Println("dot detected!", file, fr, pattern, pr)
						break
					}

					// ** swallows a segment, and continue.
					m.log.Println("globstar swallow a segment, and continue")
					fr++
				}
			}

			// no match was found.
			// However, in partial mode, we can't say this is necessarily over.
			// If there's more *pattern* left, then
			if partial {
				// ran out of file
				m.log.Println("\n>>> no match, partial?", file, fr, pattern, pr)
				if fr == fl {
					return true
				}
			}

			return false
		}

		// something other than **
		// non-magic patterns just have to match exactly
		// patterns with magic have been turned into regexps.
		hit := p.MatchString(f)
		m.log.Println("pattern match", p, f, hit)
		if !hit {
			return false
		}
	}

	// Note: ending in / means that we'll get a final ""
	// at the end of the pattern.  This can only match a
	// corresponding "" at the end of the file.
	// If the file ends in /, then it can only match a
	// a pattern that ends in /, unless the pattern just
	// doesn't have any more for it. But, a/b/ should *not*
	// match "a/b/*", even though "" matches against the
	// [^/]*? pattern, except in partial mode, where it might
	// simply not be reached yet.
	// However, a/b/ should still satisfy a/*

	// now either we fell off the end of the pattern, or we're done.
	if fi == fl && pi == pl {
		// ran out of pattern and filename at the same time.
		// an exact hit!
		return true
	} else if fi == fl {
		// ran out of file, but still had pattern left.
		// this is ok if we're doing the match as part of
		// a glob fs traversal.
		return partial
	} else if pi == pl {
		// ran out of pattern, still have file left.
		// this is only acceptable if we're on the very last
		// empty segment of a file with a trailing slash.
		// a/* should match a/b/
		return fi == fl-1 && file[fi] == ""
	}

	// should be unreachable.
	panic("wtf?")
}
