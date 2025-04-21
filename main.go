package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
)

func printTree(node *Node, prefix string, isLeft bool) {
	if node == nil {
		return
	}

	if !isLeft {
		fmt.Print(prefix + "└─R-")
		prefix += "   "
	} else {
		fmt.Print(prefix + "├─L-")
		prefix += "│  "
	}

	if node.Char == 0 {
		fmt.Printf("(%d)\n", node.Freq)
	} else {
		fmt.Printf("%c:%d\n", node.Char, node.Freq)
	}

	printTree(node.Left, prefix, true)
	printTree(node.Right, prefix, false)
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
	Path uint64 `json:"p"`
	Size int    `json:"s"`
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
			prevLast := pairs[len(pairs)-1]
			curMainNode.Right = &prevLast
			curMainNode.Left = &node
			pairs[len(pairs)-1] = curMainNode
		}
	}

	return buildTree(pairs)
}

func huffmanEncoding(input string) ([]CharPathEncoding, map[byte]CharPathEncoding) {
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
	printTree(asTree, "", false)
	charDict := map[byte]CharPathEncoding{}
	treeToDict(asTree, charDict, &Path{})

	encoded := []CharPathEncoding{}
	for _, bt := range asBts {
		encoded = append(encoded, charDict[bt])
	}

	return encoded, charDict
}

func writeCompressionToFile(bits []CharPathEncoding, dict map[byte]CharPathEncoding, filename string) error {
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


	metadata := CompressedFileMetaData{
		EncodedLen:  binBuf.Len(),
		Dict:        dict,
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
			return 0, err
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

func decodeCompressedFile(filename string) (string, error) {
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
	binData := []byte{}
	readMeta := true
	for _, bt := range content {
		if bt == META_SEPARATOR {
			readMeta = false
			continue
		}

		if readMeta {
			metaBtsRead = append(metaBtsRead, bt)
		} else {
			binData = append(binData, bt)
		}

	}

	for i, b := range binData {
		fmt.Printf("Byte %d: %08b\n", i, b)
	}

	metaR := CompressedFileMetaData{}
	err = json.Unmarshal(metaBtsRead, &metaR)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal meta bytes: %+v", err)
	}

	binBuf := bytes.NewBuffer(binData)
	binReader := newBitReader(binBuf, int64(metaR.EncodedLen), metaR.PaddingSize)

	fmt.Println("dict:")
	for char, encoding := range metaR.Dict {
		fmt.Printf("Char: %c, Code: ", char)
		for i := encoding.Size - 1; i >= 0; i-- {
			bit := (encoding.Path >> uint(i)) & 1
			fmt.Printf("%d", bit)
		}

		fmt.Printf(" (Size: %d, Raw Path: %d)\n", encoding.Size, encoding.Path)
	}

	lookupTable := map[CharPathEncoding]byte{}
	for key, val := range metaR.Dict {
		lookupTable[val] = key
	}

	currPath := uint64(0)
	var pathLen int
	var decoded string

	for binReader.Next() {
		bit, err := binReader.ReadBit()
		if err != nil {
			return "", err
		}

		// e.g., if our bit is 1 and our currentPath is 0 currentPath = 1
		currPath = (currPath << 1) | uint64(bit)
		pathLen++

		debugStr := ""
		for i := 0; i < pathLen; i++ {
			bit := (currPath >> uint(pathLen-1-i)) & 1
			debugStr += fmt.Sprintf("%d", bit)
		}

		if char, ok := lookupTable[CharPathEncoding{Path: currPath, Size: pathLen}]; ok {
			fmt.Println("debugStr:", debugStr)
			fmt.Println("char found: ", string(char))
			decoded += string(char)
			pathLen = 0
			currPath = 0
		}
	}

	return decoded, nil
}

type CompressedFileMetaData struct {
	EncodedLen  int                       `json:"e"`
	Dict        map[byte]CharPathEncoding `json:"d"`
	PaddingSize int                       `json:"ps"`
}

const META_SEPARATOR = '#'

func main() {
	filename := "data"

	encodeStr := "hello world! From Emil."
	encoded, charDict := huffmanEncoding(encodeStr)

	err := writeCompressionToFile(encoded, charDict, filename)
	if err != nil {
		panic(err)
	}

	decoded, err := decodeCompressedFile(filename)
	if err != nil {
		panic(err)
	}

	fmt.Printf("decoded: %s\n", decoded)
	fmt.Println(len(decoded))

}
