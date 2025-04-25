package mft

import (
	"slices"
)

const BYTE_LMT = 256

func Mft(input []byte) []byte {
	allBytes := make([]byte, BYTE_LMT)
	for idx := 0; idx < BYTE_LMT; idx++ {
		allBytes[idx] = byte(idx)
	}

	result := make([]byte, len(input))

	for idx, bt := range input {
		btIdx := slices.Index(allBytes, bt)
		result[idx] = byte(btIdx)

		if btIdx > 0 {
			allBytes = slices.Delete(allBytes, btIdx, btIdx+1)
			allBytes = slices.Insert(allBytes, 0, bt)
		}
	}

	return result
}

func DecodeMft(input []byte) []byte {
	allBytes := make([]byte, BYTE_LMT)
	for idx := 0; idx < BYTE_LMT; idx++ {
		allBytes[idx] = byte(idx)
	}

	result := make([]byte, len(input))

	for idx, pos := range input {
		b := allBytes[pos]
		result[idx] = b

		if pos > 0 {
			allBytes = slices.Delete(allBytes, int(pos), int(pos)+1)
			allBytes = slices.Insert(allBytes, 0, b)
		}
	}

	return result
}
