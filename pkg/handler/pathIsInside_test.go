package handler

import (
	"testing"
)

func TestUnix(t *testing.T) {
	type pathTest = struct {
		a, b   string
		expect bool
	}

	nixTests := []pathTest{
		{"/x/y/z", "/a/b/c", false},
		{"/x/y/z", "/x/y", true},
		{"/x/y/z", "/x/y/z", true},
		{"/x/y/z", "/x/y/z/w", false},
		{"/x/y/z", "/x/y/w", false},

		{"/x/y", "/x/yy", false},
		{"/x/yy", "/x/y", false},

		{"/X/y/z", "/x/y", false},
		{"/x/Y/z", "/x/y/z", false},
	}

	for _, item := range nixTests {
		if pathIsInside(item.a, item.b) != item.expect {
			t.Errorf("pathIsInside(%s, %s) != %v", item.a, item.b, item.expect)
		}
		if pathIsInside(item.a+"/", item.b) != item.expect {
			t.Errorf("pathIsInside(%s/, %s) != %v", item.a, item.b, item.expect)
		}
		if pathIsInside(item.a, item.b+"/") != item.expect {
			t.Errorf("pathIsInside(%s, %s/) != %v", item.a, item.b, item.expect)
		}
		if pathIsInside(item.a+"/", item.b+"/") != item.expect {
			t.Errorf("pathIsInside(%s/, %s/) != %v", item.a, item.b, item.expect)
		}
	}
}

func xTestWindows(t *testing.T) {
	type pathTest = struct {
		a, b   string
		expect bool
	}

	winTests := []pathTest{
		{"C:\\x\\y\\z", "C:\\a\\b\\c", false},
		{"C:\\x\\y\\z", "C:\\x\\y", true},
		{"C:\\x\\y\\z", "C:\\x\\y\\z", true},
		{"C:\\x\\y\\z", "C:\\x\\y\\z\\w", false},
		{"C:\\x\\y\\z", "C:\\x\\y\\w", false},

		{"C:\\x\\y", "C:\\x\\yy", false},
		{"C:\\x\\yy", "C:\\x\\y", false},

		{"C:\\x\\y\\z", "D:\\x\\y\\z", false},
		{"C:\\x\\y\\z", "D:\\x\\y\\z\\w", false},
	}

	for _, item := range winTests {
		if pathIsInside(item.a, item.b) != item.expect {
			t.Errorf("pathIsInside(%s, %s) != %v", item.a, item.b, item.expect)
		}
		if pathIsInside(item.a+"\\", item.b) != item.expect {
			t.Errorf("pathIsInside(%s\\, %s) != %v", item.a, item.b, item.expect)
		}
		if pathIsInside(item.a, item.b+"\\") != item.expect {
			t.Errorf("pathIsInside(%s, %s\\) != %v", item.a, item.b, item.expect)
		}
		if pathIsInside(item.a+"\\", item.b+"\\") != item.expect {
			t.Errorf("pathIsInside(%s\\, %s\\) != %v", item.a, item.b, item.expect)
		}
	}
}
