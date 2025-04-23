package stinkycompressor

import (
	"fmt"
	"os"
	"testing"
)

func helperDeleteFile(t *testing.T, filename string) {
	err := os.Remove(filename)
	if err != nil {
		t.Fatalf("failed to delete file: %+v", err)
	}
}

func TestCanEncodeAndDecodeFromFile(t *testing.T) {
	runTimes := 1000
	for idx := 0; idx < runTimes; idx++ {
		t.Run(fmt.Sprintf("run-%d", idx), func(t *testing.T) {
			testFileName := fmt.Sprintf("test-file-%d", idx)
			defer helperDeleteFile(t, fmt.Sprintf("%s.%s", testFileName, COMPRESSED_FILE_EXTENSION))
			input := "The ancient oak tree stood as a silent sentinel at the edge of the meadow, its gnarled branches reaching skyward like arthritic fingers. Generation after generation had sought shelter beneath its broad canopy, from summer picnics to winter storms. Children had climbed its sturdy limbs, lovers had carved their initials into its weathered bark, and birds had built countless nests among its leaves. Through drought and flood, through war and peace, the tree remained a living testament to resilience and time. Locals claimed it was over three hundred years old, though no one knew for certain. What was known, however, was that the oak had become more than just a tree; it had become a landmark, a meeting place, a character in the story of the town itself. bobs burgers and fried."

			asBytes := []byte(input)
			compressedFileName, err := WriteCompressionToFile(asBytes, testFileName, false, false)
			if err != nil {
				t.Fatalf("writeCompressionToFile: %+v", err)
			}

			decoded, err := DecodeCompressedFile(compressedFileName, false)
			if err != nil {
				t.Fatalf("decodeCompressedFile: %+v", err)
			}

			if decoded != input {
				t.Fatalf("decoded message did not match input.\nWanted: %s\nGot: %s", input, decoded)
			}
		})
	}
}
