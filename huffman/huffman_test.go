package huffman

import (
	"fmt"
	"reflect"
	"stinky-compression/file"
	"testing"
)

func TestCanEncodeAndDecodeStringCorrectly(t *testing.T) {
	runTimes := 1000
	for idx := 0; idx < runTimes; idx++ {
		t.Run(fmt.Sprintf("long-input-%d", idx), func(t *testing.T) {
			input := "The-ancient-oak tree stood as a silent sentinel at the edge of the meadow, its gnarled branches reaching skyward like arthritic fingers. Generation after generation had sought shelter beneath its broad canopy, from summer picnics to winter storms. Children had climbed its sturdy limbs, lovers had carved their initials into its weathered bark, and birds had built countless nests among its leaves. Through drought and flood, through war and peace, the tree remained a living testament to resilience and time. Locals claimed it was over three hundred years old, though no one knew for certain. What was known, however, was that the oak had become more than just a tree; it had become a landmark, a meeting place, a character in the story of the town itself."
			asBytes := []byte(input)

			encoded, dict, bwtIdx, rleDict := HuffmanEncoding(asBytes, false)
			decoded := DecodeCompressionFromTable(encoded, dict, bwtIdx, rleDict)

			if len(decoded) != len(asBytes) {
				t.Fatalf("decoded bytes len did not match input bytes len, got %d, wanted %d", len(decoded), len(asBytes))
			}

			if string(decoded) != string(asBytes) {
				t.Fatalf("decoded did not match input got\n%s\nwanted\n%s", string(decoded), string(asBytes))
			}
		})
	}
}

func TestCanEncodeAndDecodeBinaryCorrectly(t *testing.T) {
	bts, err := file.ReadInputFile("./testdata/input.jpeg")
	if err != nil {
		t.Fatalf("failed to read input file: %s", err.Error())
	}

	encoded, dict, bwtIdx, rleDict := HuffmanEncoding(bts, false)
	decoded := DecodeCompressionFromTable(encoded, dict, bwtIdx, rleDict)

	if len(decoded) != len(bts) {
		t.Fatalf("decoded bytes len did not match input bytes len, got %d, wanted %d\n", len(decoded), len(bts))
	}

	deepEqual := reflect.DeepEqual(decoded, bts)
	if !deepEqual {
		numInEqual := 0
		uniqueBts := map[byte]bool{}
		for idx, bt := range bts {
			if decoded[idx] != bt {
				numInEqual += 1
				uniqueBts[bt] = true
			}
		}

		uniqueBytsList := []byte{}
		for key := range uniqueBts {
			uniqueBytsList = append(uniqueBytsList, key)
		}

		t.Fatalf("%d/%d of decoded bytes did not match input bts\n these bytes did not match %+v\n", numInEqual, len(decoded), uniqueBytsList)
	}

}
