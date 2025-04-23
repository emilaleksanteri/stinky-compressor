package main

import (
	"fmt"
	"os"
	stinkycompressor "stinky-compression/stinky-compressor"
)

func main() {
	filename := "data-big"
	debug := false
	removeOldFile := false

	encodeStr := "The ancient oak tree stood as a silent sentinel at the edge of the meadow, its gnarled branches reaching skyward like arthritic fingers. Generation after generation had sought shelter beneath its broad canopy, from summer picnics to winter storms. Children had climbed its sturdy limbs, lovers had carved their initials into its weathered bark, and birds had built countless nests among its leaves. Through drought and flood, through war and peace, the tree remained a living testament to resilience and time. Locals claimed it was over three hundred years old, though no one knew for certain. What was known, however, was that the oak had become more than just a tree; it had become a landmark, a meeting place, a character in the story of the town itself. Bobs burgers and fries."
	bytes := []byte(encodeStr)

	compressedFileName, err := stinkycompressor.WriteCompressionToFile(bytes, filename, removeOldFile, debug)
	if err != nil {
		panic(err)
	}

	decoded, err := stinkycompressor.DecodeCompressedFile(compressedFileName, debug)
	if err != nil {
		panic(err)
	}

	fmt.Printf("input:\n'%s'\n", encodeStr)
	fmt.Printf("decoded:\n'%s'\n", decoded)
	fmt.Printf("are equal: %v\n", encodeStr == decoded)
	fmt.Println(len(decoded), len(encodeStr))
	if len(decoded) == len(encodeStr) {
		for idx, char := range encodeStr {
			match := decoded[idx]
			if match != byte(char) {
				fmt.Printf("mismatched char at idx %d: '%s', wanted '%s'\n", idx, string(match), string(byte(char)))
			}
		}
	}

	err = os.Remove(compressedFileName)
	if err != nil {
		panic(fmt.Sprintf("failed to delete file: %+v", err))
	}
}
