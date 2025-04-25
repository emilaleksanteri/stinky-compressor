package bwt

import "slices"

const PRIMARY_INDEX_MARKER = byte('%')

type bwtRotation struct {
	offset int
	data   []byte
}

func Bwt(input []byte) ([]byte, int) {
	if len(input) == 0 {
		return []byte{}, 0
	}

	processBts := []byte{PRIMARY_INDEX_MARKER}
	processBts = append(processBts, input...)

	size := len(processBts)
	rotations := make([]bwtRotation, size)
	for idx := 0; idx < size; idx++ {
		rotations[idx] = bwtRotation{
			offset: idx,
			data:   processBts,
		}
	}

	slices.SortFunc(rotations, func(a, b bwtRotation) int {
		for k := 0; k < size; k++ {
			ca := a.data[(a.offset+k)%size]
			cb := b.data[(b.offset+k)%size]

			if ca != cb {
				if ca < cb {
					return -1
				}

				return 1
			}
		}

		return 0
	})

	primaryIdx := 0
	for idx, rot := range rotations {
		if rot.offset == 0 {
			primaryIdx = idx
			break
		}
	}

	result := make([]byte, size)
	for idx, rot := range rotations {
		lastCharIdx := (rot.offset + size - 1) % size
		result[idx] = rot.data[lastCharIdx]
	}

	return result, primaryIdx
}

type bwtPair struct {
	char byte
	idx  int
}

func DecodeBwt(data []byte, primaryIdx int) []byte {
	size := len(data)

	firstCol := make([]bwtPair, size)
	for idx := 0; idx < size; idx++ {
		firstCol[idx] = bwtPair{char: data[idx], idx: idx}
	}

	slices.SortFunc(firstCol, func(a, b bwtPair) int {
		if a.char != b.char {
			if a.char < b.char {
				return -1
			}

			return 1
		}

		if a.idx < b.idx {
			return -1
		}

		if a.idx > b.idx {
			return 1
		}

		return 0
	})

	result := make([]byte, size)
	row := primaryIdx
	for idx := 0; idx < size; idx++ {
		char := firstCol[row].char
		result[idx] = char
		row = firstCol[row].idx
	}

	return result[1:]
}
