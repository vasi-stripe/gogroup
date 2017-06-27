package gogroup

import "strings"

// Group everything together.
type grouperCombined struct{}

func (grouperCombined) Group(pkgPath string) (group int) {
	return 0
}

// Group like goimports: first standard packages, then 3rd party, then appengine,
// then local.
type grouperGoimports struct{}

func (grouperGoimports) Group(pkgPath string) (group int) {
	if strings.HasPrefix(pkgPath, "local/") {
		return 3
	} else if strings.HasPrefix(pkgPath, "appengine") {
		return 2
	} else if strings.Contains(pkgPath, ".") {
		return 1
	} else {
		return 0
	}
}

// Group with another common pattern: std, local, 3rd party.
type grouperLocalMiddle struct{}

func (grouperLocalMiddle) Group(pkgPath string) (group int) {
	if strings.HasPrefix(pkgPath, "local/") {
		return 1
	} else if strings.Contains(pkgPath, ".") {
		return 2
	} else {
		return 0
	}
}

// You could group in really strange ways. I guess it's supported?
type grouperWeird struct{}

func (grouperWeird) Group(pkgPath string) (group int) {
	return strings.Count(pkgPath, "/")
}
