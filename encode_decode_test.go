package main

import (
	"fmt"
	"os"
	"testing"
)

func TestCanEncodeAndDecodeStringCorrectly(t *testing.T) {
	input := "The quick brown fox jumps over the lazy dog"
	encoded, dict := huffmanEncoding(input)
	decoded := decodeHuffman(encoded, dict)

	if decoded != input {
		t.Fatalf("decoded message did not match input.\nWanted: %s\nGot: %s", input, decoded)
	}
}

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
			defer helperDeleteFile(t, testFileName)
			input := "The quick brown fox jumps over the lazy dog"
			encoded, dict := huffmanEncoding(input)

			err := writeCompressionToFile(encoded, dict, testFileName)
			if err != nil {
				t.Fatalf("writeCompressionToFile: %+v", err)
			}

			decoded, err := decodeCompressedFile(testFileName)
			if err != nil {
				t.Fatalf("decodeCompressedFile: %+v", err)
			}

			if decoded != input {
				t.Fatalf("decoded message did not match input.\nWanted: %s\nGot: %s", input, decoded)
			}
		})
	}
}
