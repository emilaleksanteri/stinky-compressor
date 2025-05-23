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

func TestCanEncodeAndDecodeBWTWithLongInput(t *testing.T) {
	input := "The-ancient-oak tree stood as a silent sentinel at the edge of the meadow, its gnarled branches reaching skyward like arthritic fingers. Generation after generation had sought shelter beneath its broad canopy, from summer picnics to winter storms. Children had climbed its sturdy limbs, lovers had carved their initials into its weathered bark, and birds had built countless nests among its leaves. Through drought and flood, through war and peace, the tree remained a living testament to resilience and time. Locals claimed it was over three hundred years old, though no one knew for certain. What was known, however, was that the oak had become more than just a tree; it had become a landmark, a meeting place, a character in the story of the town itself."

	asBts := []byte(input)
	encoded, pIndex := Bwt(asBts)
	t.Log(string(encoded))

	decoded := DecodeBwt(encoded, pIndex)

	if string(decoded) != input {
		t.Fatalf("decoded did not match input, got\n%s\nwanted\n%s\n", string(decoded), string(input))
	}

}
