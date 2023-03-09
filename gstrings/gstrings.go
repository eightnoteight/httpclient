package gstrings

import "strings"

type StringLike interface {
	~string
}

func Join[Str StringLike](s []Str, sep string) string {
	sarr := make([]string, len(s))
	for i, str := range s {
		sarr[i] = string(str)
	}
	return strings.Join(sarr, sep)
}
