package rle

func Rle(input []byte) ([]byte, []int32) {
	idxDict := []int32{}
	encoded := []byte{}

	currCharCount := 0
	countedChar := byte('0')
	for idx, bt := range input {
		if countedChar != bt {
			if idx != 0 {
				idxDict = append(idxDict, int32(currCharCount))
				encoded = append(encoded, countedChar)
			}

			currCharCount = 0
			countedChar = bt
		}

		currCharCount++
	}

	if currCharCount != 0 {
		encoded = append(encoded, countedChar)
		idxDict = append(idxDict, int32(currCharCount))
	}

	return encoded, idxDict
}

func DecodeRle(input []byte, decodeDict []int32) []byte {
	if len(decodeDict) == 0 {
		return input
	}

	decoded := []byte{}
	for idx, char := range input {
		count := decodeDict[idx]
		for c := 0; c < int(count); c++ {
			decoded = append(decoded, char)
		}
	}

	return decoded
}
