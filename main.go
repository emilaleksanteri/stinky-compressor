package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

func printTree(node *Node, prefix string, isLeft bool, isFirst bool) {
	if node == nil {
		return
	}

	if isLeft && !isFirst {
		fmt.Print(prefix + "├─L-")
		prefix += "│  "
	} else if !isLeft && !isFirst {
		fmt.Print(prefix + "└─R-")
		prefix += "   "
	}

	if node.Char == 0 {
		fmt.Printf("(%d)\n", node.Freq)
	} else {
		fmt.Printf("%c:%d\n", node.Char, node.Freq)
	}

	printTree(node.Left, prefix, true, false)
	printTree(node.Right, prefix, false, false)
}

type Path struct {
	path  uint64
	depth int
}

func (p *Path) Left() {
	p.path <<= 1
	p.depth += 1
}

func (p *Path) Right() {
	p.path <<= 1
	p.path |= 1
	p.depth += 1
}

func (p *Path) Up() {
	if p.depth > 0 {
		p.path >>= 1
		p.depth -= 1
	}
}

// to represent 0bxxx where x is the binary num
func treeToDict(node *Node, dict map[byte]CharPathEncoding, path *Path) {
	if node == nil {
		return
	}

	if node.Left == nil && node.Right == nil {
		dict[node.Char] = CharPathEncoding{
			Path: path.path,
			Size: path.depth,
			Freq: node.Freq,
		}
	}

	path.Left()
	treeToDict(node.Left, dict, path)
	path.Up()

	path.Right()
	treeToDict(node.Right, dict, path)
	path.Up()

}

type Node struct {
	Char  byte
	Freq  int
	Left  *Node
	Right *Node
}

type CharEncoding struct {
	Val  byte
	Freq int
}

type CharPathEncoding struct {
	Path     uint64 `json:"-"`
	PathMeta string `json:"p"`
	Size     int    `json:"s"`
	Freq     int    `json:"f"`
}

func (c *CharPathEncoding) MakeSafePathMeta() {
	binStr := ""
	for pos := c.Size - 1; pos >= 0; pos-- {
		bit := (c.Path >> uint(pos)) & 1
		if bit == 0 {
			binStr += "0"
		} else {
			binStr += "1"
		}
	}

	c.PathMeta = binStr
}
func (c *CharPathEncoding) DecodeSafePathMetaToPath() {
	var bin uint64
	for _, char := range c.PathMeta {
		bin <<= 1 // shift by 1
		if char == '1' {
			bin |= 1 // flip 0 to 1
		}
	}

	c.Path = bin
}

func treeFromDict(dict map[string]CharPathEncoding) *Node {
	nodes := []Node{}
	for char, enc := range dict {
		nodes = append(nodes, Node{
			Char: []byte(char)[0],
			Freq: enc.Freq,
		})
	}

	/*
		Think what can be done is first sort by path where longest paths come first
		then we just build, if the last bit of the path is 0 it goes to the parent nodes left, else to the right
		and do this till all the nodes are made into the tree

		then to triverse we just travel until we find a letter, if nothing found in path then we must not have a char yet

	*/

	slices.SortFunc(nodes, func(a, b Node) int {
		aEnc := dict[string(a.Char)]
		bEnc := dict[string(b.Char)]

		if aEnc.Size > bEnc.Size {
			return -1
		}

		if aEnc.Size < bEnc.Size {
			return 1
		}

		return 0
	})

	// Since first 4 nodes will all have the same size (being part of the last big node), they have to be sorted by the last 2 bits of their path
	// 11 ending means that it will be the absolute last char in our tree and 00 means it will be the first one out of the 4 last ones
	// since we are building the tree from last node to first node, the absolute last needs to be in the front of the node list etc etc
	firstFour := nodes[:4]

	codeCompMap := map[uint64]int{
		0b11: 4,
		0b10: 3,
		0b01: 2,
		0b00: 1,
	}

	slices.SortFunc(firstFour, func(a, b Node) int {
		aEnc := dict[string(a.Char)]
		mask := uint64(3)
		aLastBts := aEnc.Path & mask

		bEnc := dict[string(b.Char)]
		bLastBts := bEnc.Path & mask

		aScore := codeCompMap[aLastBts]
		bScore := codeCompMap[bLastBts]

		if aScore > bScore {
			return -1
		}

		if aScore < bScore {
			return 1
		}

		return 0
	})

	for idx, node := range firstFour {
		nodes[idx] = node
	}

	currNode := Node{}
	currAccNode := Node{}
	hasRemainingNonPair := false

	for idx, node := range nodes {
		lastBit := (dict[string(node.Char)].Path >> uint(0)) & 1
		if lastBit == 0 {
			currAccNode.Left = &node
		} else {
			currAccNode.Right = &node
		}

		if currAccNode.Left != nil && currAccNode.Right != nil {
			currAccNode.Freq = currAccNode.Left.Freq + currAccNode.Right.Freq

			if currNode.Right == nil {
				saved := currAccNode
				currNode.Right = &saved
				currAccNode = Node{}
			} else if currNode.Left == nil {
				saved := currAccNode
				currNode.Left = &saved
				currAccNode = Node{}
			}

			if currNode.Left != nil && currNode.Right != nil {
				saved := currNode
				saved.Freq = saved.Left.Freq + saved.Right.Freq

				currNode = Node{}
				currNode.Right = &saved
			}

		} else if idx == len(nodes)-1 && currNode.Left == nil && (currAccNode.Left == nil || currAccNode.Right == nil) {
			currNode.Left = &node
			currNode.Freq = currNode.Left.Freq + currNode.Right.Freq
			hasRemainingNonPair = true

		}

	}

	if hasRemainingNonPair {
		return &currNode
	}

	return currNode.Right
}

func buildTree(pairs []Node) *Node {
	currHead := Node{}
	for idx, pair := range pairs {
		if currHead.Right == nil {
			currHead.Right = &pair
		} else {
			currHead.Left = &pair
		}

		if currHead.Right != nil && currHead.Left != nil {
			currHead.Freq = currHead.Right.Freq + currHead.Left.Freq
			if idx == len(pairs)-1 {
				continue
			}

			saved := currHead
			currHead = Node{}
			currHead.Right = &saved
		}

	}

	return &currHead
}

func encodeToTree(chars []CharEncoding) *Node {
	pairs := []Node{}

	curMainNode := Node{}
	for idx, char := range chars {
		node := Node{
			Char: char.Val,
			Freq: char.Freq,
		}

		if curMainNode.Right == nil {
			curMainNode.Right = &node
		} else {
			curMainNode.Left = &node
		}

		if curMainNode.Left != nil && curMainNode.Right != nil {
			curMainNode.Freq = curMainNode.Left.Freq + curMainNode.Right.Freq
			pairs = append(pairs, curMainNode)
			curMainNode = Node{}
		} else if idx == len(chars)-1 && (curMainNode.Left == nil || curMainNode.Right == nil) {
			pairs = append(pairs, node)
		}
	}

	return buildTree(pairs)
}

func huffmanEncoding(input string, debugMode bool) ([]CharPathEncoding, map[byte]CharPathEncoding) {
	asBts := []byte(input)

	occurance := map[byte]int{}
	for _, bt := range asBts {
		if _, ok := occurance[bt]; ok {
			occurance[bt] += 1
		} else {
			occurance[bt] = 1
		}
	}

	asList := []CharEncoding{}
	for key, val := range occurance {
		asList = append(asList, CharEncoding{
			Val:  key,
			Freq: val,
		})
	}

	slices.SortFunc(asList, func(a, b CharEncoding) int {
		if a.Freq > b.Freq {
			return 1
		}

		if a.Freq < b.Freq {
			return -1
		}

		return 0
	})

	asTree := encodeToTree(asList)
	if debugMode {
		printTree(asTree, "", false, true)
	}
	charDict := map[byte]CharPathEncoding{}
	treeToDict(asTree, charDict, &Path{})

	encoded := []CharPathEncoding{}
	for _, bt := range asBts {
		encoded = append(encoded, charDict[bt])
	}

	return encoded, charDict
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	return false
}

func writeCompressionToFile(bits []CharPathEncoding, dict map[byte]CharPathEncoding, filename string) error {
	if !fileExists(filename) {
		fileC, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file: %+v", err)
		}
		fileC.Close()
	}

	file, err := os.OpenFile(filename, os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("opening file failed: %+v", err)
	}
	defer file.Close()

	buf := []byte{}
	binBuf := bytes.NewBuffer(buf)
	binWriter := newBitWriter(binBuf)
	for _, enc := range bits {
		err := binWriter.WriteBits(enc.Path, enc.Size)
		if err != nil {
			return fmt.Errorf("failed to write bits: %+v", err)
		}

	}

	padding, err := binWriter.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush bits: %+v", err)
	}

	metaMap := map[string]CharPathEncoding{}
	for ch, enc := range dict {
		enc.MakeSafePathMeta()
		metaMap[string(ch)] = enc
	}

	metadata := CompressedFileMetaData{
		EncodedLen:  binBuf.Len(),
		Dict:        metaMap,
		PaddingSize: padding,
	}

	metaBts, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %+v", err)
	}

	metaBts = append(metaBts, byte(META_SEPARATOR))
	_, err = file.Write(metaBts)
	if err != nil {
		return fmt.Errorf("failed to write meta bytes to file: %+v", err)
	}

	_, err = binBuf.WriteTo(file)
	if err != nil {
		return fmt.Errorf("failed to write bits to file: %+v", err)
	}

	return nil
}

func decodeHuffman(bits []CharPathEncoding, dict map[byte]CharPathEncoding) string {
	decoded := ""
	for _, u := range bits {
		for key, val := range dict {
			if val.Path == u.Path {
				decoded += string(key)
				break
			}
		}
	}

	return decoded
}

type BitWriter struct {
	writer      io.Writer
	buffer      byte // we write 8 bits at a time which is a byte
	bitsWritten int
}

func newBitWriter(w io.Writer) *BitWriter {
	return &BitWriter{
		writer: w,
	}
}

func (b *BitWriter) WriteBits(path uint64, bitSize int) error {
	for pos := bitSize - 1; pos >= 0; pos-- {
		bit := (path >> uint(pos)) & 1 // gets first bit in path i.e. if path is 01001 we would get 0 at first iter since we right shift by size - 1

		// if we have 111 we shift L shift by one making it 1110, then with | op we append our bit to the L shiften position,
		// so if our bit is 1 b.buffer would become 1111
		b.buffer = (b.buffer << 1) | byte(bit)

		b.bitsWritten++ // keep track to see when we have 8 bits or a full byte

		if b.bitsWritten == 8 {
			_, err := b.writer.Write([]byte{b.buffer})
			if err != nil {
				return err
			}

			b.buffer = 0
			b.bitsWritten = 0
		}
	}

	return nil
}

// returned int is a encoding padding size which will be needed when decoding the file
func (b *BitWriter) Flush() (int, error) {
	paddingSize := 0
	if b.bitsWritten > 0 {
		// since we can only write 8 bits at a time, if the remaining bits are not exactly a byte, we need to add some padding
		// to make a full byte
		paddingSize = 8 - b.bitsWritten
		b.buffer <<= byte(paddingSize)

		_, err := b.writer.Write([]byte{b.buffer})
		if err != nil {
			return paddingSize, err
		}

		b.buffer = 0
		b.bitsWritten = 0

	}

	return paddingSize, nil
}

type BitReader struct {
	reader         io.Reader
	buffer         byte // 8 bits at a time
	bitsRemaining  int
	paddingSize    int
	totalBytesRead int64
	encodedSize    int64
	finished       bool
}

func (b *BitReader) Next() bool {
	if b.finished {
		return false
	}

	return true
}

func newBitReader(reader io.Reader, encodedSize int64, paddingSize int) *BitReader {
	return &BitReader{
		reader:        reader,
		bitsRemaining: 0,
		paddingSize:   paddingSize,
		encodedSize:   encodedSize,
	}
}

func (b *BitReader) ReadBit() (byte, error) {
	if b.bitsRemaining == 0 {
		if b.totalBytesRead >= b.encodedSize {
			b.finished = true
			return 0, nil
		}

		buf := make([]byte, 1)
		_, err := b.reader.Read(buf)
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				b.finished = true
				return 0, nil
			default:
				return 0, err
			}
		}

		b.buffer = buf[0]
		b.bitsRemaining = 8
		b.totalBytesRead++

		if b.totalBytesRead == b.encodedSize {
			// we substract padding from our buffer expected read size for the case that we have something like
			// 11110000 with our buffer being 4 bits in this example
			// now, we can still use our grab first bit method by total size offset below
			// but we stop reading before we get to the 0000 since our bitsRemaining is 4 instead of 8

			b.bitsRemaining -= b.paddingSize
			if b.bitsRemaining <= 0 {
				b.finished = true
				return 0, nil
			}
		}
	}

	// our buffer is always 8 bits, what we do in the two lines here is we grab the first bit of the buffer
	// once we have that bit we left shift the buffer to remove the first bit and move everything forward by 1
	// as we append a 0 bit to the end
	// e.g., if we have 11111111 as our buffer bits, we do this:
	// 1 . take 1 from the front
	// 2. left shift by 1 and our buffer becomes 11111110
	bit := (b.buffer >> 7) & 1
	b.buffer <<= 1

	b.bitsRemaining--

	return bit, nil
}

func decodeCompressedFile(filename string, debug bool) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %+v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %+v", err)
	}

	metaBtsRead := []byte{}
	readMeta := true
	metaEndsIdx := 0

	for idx, bt := range content {
		if bt == META_SEPARATOR {
			readMeta = false
			metaEndsIdx = idx + 1
			break
		}

		if readMeta {
			metaBtsRead = append(metaBtsRead, bt)
		}
	}

	metaR := CompressedFileMetaData{}
	err = json.Unmarshal(metaBtsRead, &metaR)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal meta bytes: %+v", err)
	}

	binData := make([]byte, metaR.EncodedLen)
	contentLen := len(content)
	binDataIdx := 0
	for {
		if metaEndsIdx == contentLen {
			break
		}

		binData[binDataIdx] = content[metaEndsIdx]
		binDataIdx++
		metaEndsIdx++
	}

	for ch, enc := range metaR.Dict {
		enc.DecodeSafePathMetaToPath()
		metaR.Dict[ch] = enc
	}

	binBuf := bytes.NewBuffer(binData)
	binReader := newBitReader(binBuf, int64(metaR.EncodedLen), metaR.PaddingSize)

	tree := treeFromDict(metaR.Dict)
	if debug {
		printTree(tree, "", false, true)
	}

	var decoded string

	head := tree

	for binReader.Next() {
		bit, err := binReader.ReadBit()
		if err != nil {
			return "", fmt.Errorf("read bit: %+v", err)
		}

		if bit == 1 {
			head = head.Right
		} else {
			head = head.Left
		}

		if head.Left == nil && head.Right == nil {
			decoded += string(head.Char)
			head = tree
		}
	}

	return decoded, nil
}

type CompressedFileMetaData struct {
	EncodedLen  int                         `json:"e"`
	Dict        map[string]CharPathEncoding `json:"d"`
	PaddingSize int                         `json:"ps"`
}

const META_SEPARATOR = '#'

func main() {
	filename := "data-big"

	encodeStr := "The ancient oak tree stood as a silent sentinel at the edge of the meadow, its gnarled branches reaching skyward like arthritic fingers. Generation after generation had sought shelter beneath its broad canopy, from summer picnics to winter storms. Children had climbed its sturdy limbs, lovers had carved their initials into its weathered bark, and birds had built countless nests among its leaves. Through drought and flood, through war and peace, the tree remained a living testament to resilience and time. Locals claimed it was over three hundred years old, though no one knew for certain. What was known, however, was that the oak had become more than just a tree; it had become a landmark, a meeting place, a character in the story of the town itself. Bobs burgers and fries."
	encoded, charDict := huffmanEncoding(encodeStr, true)

	err := writeCompressionToFile(encoded, charDict, filename)
	if err != nil {
		panic(err)
	}

	decoded, err := decodeCompressedFile(filename, true)
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

	err = os.Remove(filename)
	if err != nil {
		panic(fmt.Sprintf("failed to delete file: %+v", err))
	}
}
