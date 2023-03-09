package integers

import (
	"fmt"
)

type Integers interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int | ~int8 | ~int16 | ~int32 | ~int64
}

func ToStrings[T Integers](numarr []T) []string {
	strarr := make([]string, len(numarr))
	for i, num := range numarr {
		strarr[i] = fmt.Sprintf("%d", num)
	}
	return strarr
}
