package bwt

import "testing"

func TestCanEncodeAndDecodeBWT(t *testing.T) {
	input := "my favourite food is bananas"
	asBts := []byte(input)
	encoded, pIndex := Bwt(asBts)
	t.Log(string(encoded))

	decoded := DecodeBwt(encoded, pIndex)

	if string(decoded) != input {
		t.Fatalf("decoded did not match input, got\n%s\nwanted\n%s\n", string(decoded), string(input))
	}
}
