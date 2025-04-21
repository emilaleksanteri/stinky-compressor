package main

import "testing"

func TestCanEncodeAndDecodeStringCorrectly(t *testing.T) {
	input := "The quick brown fox jumps over the lazy dog"
	encoded, dict := huffmanEncoding(input)
	decoded := decodeHuffman(encoded, dict)

	if decoded != input {
		t.Fatalf("decoded message did not match input.\nWanted: %s\nGot: %s", input, decoded)
	}
}
