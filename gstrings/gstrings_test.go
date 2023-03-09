package gstrings

import (
	"testing"
)

type s1 string
type s2 = string

func TestJoin(t *testing.T) {
	// given
	sarr1 := []string{"a", "b", "c"}
	sarr2 := []s1{"a", "b", "c"}
	sarr3 := []s2{"a", "b", "c"}

	res1 := Join(sarr1, ",")
	if res1 != "a,b,c" {
		t.Errorf("expected %s, got %s", "a,b,c", res1)
	}
	res2 := Join(sarr2, ",")
	if res2 != "a,b,c" {
		t.Errorf("expected %s, got %s", "a,b,c", res2)
	}
	res3 := Join(sarr3, ",")
	if res3 != "a,b,c" {
		t.Errorf("expected %s, got %s", "a,b,c", res3)
	}
}

func TestJoinEmpty(t *testing.T) {
	// given
	sarr1 := []string{}
	sarr2 := []s1{}
	sarr3 := []s2{}

	res1 := Join(sarr1, ",")
	if res1 != "" {
		t.Errorf("expected %s, got %s", "", res1)
	}
	res2 := Join(sarr2, ",")
	if res2 != "" {
		t.Errorf("expected %s, got %s", "", res2)
	}
	res3 := Join(sarr3, ",")
	if res3 != "" {
		t.Errorf("expected %s, got %s", "", res3)
	}
}
