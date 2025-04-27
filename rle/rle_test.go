package rle

import "testing"

func TestCanRLEEncodeAndDecode(t *testing.T) {
	input := "WWWWWWWWWWWWBWWWWWWWWWWWWBBBWWWWWWWWWWWWWWWWWWWWWWWWBWWWWWWWWWWWWWW"
	asBytes := []byte(input)
	encoded, dict := Rle(asBytes)
	decoded := DecodeRle(encoded, dict)

	if len(decoded) != len(asBytes) {
		t.Fatalf("decoded bytes len did not match input bytes len, wanted %d got %d", len(asBytes), len(decoded))
	}

	if string(decoded) != input {
		t.Fatalf("decoded message did not match input\nwanted\n%s\ngot\n%s\n", input, string(decoded))
	}
}
