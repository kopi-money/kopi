package utils

import (
	"fmt"
	"strconv"
)

const padWidthSteps = 4

// PadIndex adds zeros in front of an index to make them sortable by alphabet. For example, 1 is changed to 0001, 123 is
// changed to 0123, 12345 is changed to 00012345.
func PadIndex[T int | uint64 | int64](index T) string {
	indexString := strconv.Itoa(int(index))
	indexLength := determineIndexLength(indexString)
	indexString = fmt.Sprintf(fmt.Sprintf("%%0%dd", indexLength), index)
	return indexString
}

func determineIndexLength(index string) int {
	width := len(index) - (len(index) % padWidthSteps)
	width += 1
	width *= padWidthSteps
	return width
}
