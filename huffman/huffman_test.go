package huffman

import (
	"fmt"
	"testing"
)

func TestCanEncodeAndDecodeStringCorrectly(t *testing.T) {
	runTimes := 1000
	for idx := 0; idx < runTimes; idx++ {
		t.Run(fmt.Sprintf("long-input-%d", idx), func(t *testing.T) {
			input := "The-ancient-oak tree stood as a silent sentinel at the edge of the meadow, its gnarled branches reaching skyward like arthritic fingers. Generation after generation had sought shelter beneath its broad canopy, from summer picnics to winter storms. Children had climbed its sturdy limbs, lovers had carved their initials into its weathered bark, and birds had built countless nests among its leaves. Through drought and flood, through war and peace, the tree remained a living testament to resilience and time. Locals claimed it was over three hundred years old, though no one knew for certain. What was known, however, was that the oak had become more than just a tree; it had become a landmark, a meeting place, a character in the story of the town itself."
			asBytes := []byte(input)

			encoded, dict := HuffmanEncoding(asBytes, false)
			decoded := DecodeCompressionFromTable(encoded, dict)

			if decoded != input {
				t.Fatalf("decoded message did not match input.\nWanted: %s\nGot: %s", input, decoded)
			}
		})
	}
}
