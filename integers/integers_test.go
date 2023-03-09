package integers

import "testing"

func TestToStrings(t *testing.T) {
	// given
	numarr := []int{1, 0, -1, -9, 10, -10}

	// when
	strarr := ToStrings(numarr)

	// then
	if len(strarr) != 6 {
		t.Errorf("expected %d, got %d", 6, len(strarr))
	}
	if strarr[0] != "1" {
		t.Errorf("expected %s, got %s", "1", strarr[0])
	}
	if strarr[1] != "0" {
		t.Errorf("expected %s, got %s", "0", strarr[1])
	}
	if strarr[2] != "-1" {
		t.Errorf("expected %s, got %s", "-1", strarr[2])
	}
	if strarr[3] != "-9" {
		t.Errorf("expected %s, got %s", "-9", strarr[3])
	}
	if strarr[4] != "10" {
		t.Errorf("expected %s, got %s", "10", strarr[4])
	}
	if strarr[5] != "-10" {
		t.Errorf("expected %s, got %s", "-10", strarr[5])
	}
}

func TestToStringsEmpty(t *testing.T) {
	// given
	numarr := []int{}

	// when
	strarr := ToStrings(numarr)

	// then
	if len(strarr) != 0 {
		t.Errorf("expected %d, got %d", 0, len(strarr))
	}
}
