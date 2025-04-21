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

func huffmanEncoding(input string) ([]uint64, map[byte]CharPathEncoding) {
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

	encoded := []uint64{}
	for _, bt := range asBts {
		encoded = append(encoded, charDict[bt].Path)
	}

	return encoded, charDict
}

func writeCompressionToFile(bits []uint64, dict map[byte]CharPathEncoding, filename string) error {
	file, err := os.OpenFile(filename, os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("opening file failed: %+v", err)
	}
	defer file.Close()

	toEncode := []CharPathEncoding{}
	for _, bit := range bits {
		for _, val := range dict {
			if val.Path == bit {
				toEncode = append(toEncode, val)
				break
			}
		}
	}

	padding := 0
	buf := []byte{}
	binBuf := bytes.NewBuffer(buf)
	binWriter := newBitWriter(binBuf)
	for _, enc := range toEncode {
		err := binWriter.WriteBits(enc.Path, enc.Size)
		if err != nil {
			return fmt.Errorf("failed to write bits: %+v", err)
		}

		padding, err = binWriter.Flush()
		if err != nil {
			return fmt.Errorf("failed to flush bits: %+v", err)
		}
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

	metaBts = append(metaBts, []byte(META_SEPARATOR)...)
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

func decodeHuffman(bits []uint64, dict map[byte]CharPathEncoding) string {
	decoded := ""
	for _, u := range bits {
		for key, val := range dict {
			if val.Path == u {
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
		if string(bt) == META_SEPARATOR {
			readMeta = false
			continue
		}

		if readMeta {
			metaBtsRead = append(metaBtsRead, bt)
		} else {
			binData = append(binData, bt)
		}

	}

	metaR := CompressedFileMetaData{}
	err = json.Unmarshal(metaBtsRead, &metaR)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal meta bytes: %+v", err)
	}

	asuint := []uint64{}
	for _, bin := range binData {
		asuint = append(asuint, uint64(bin))
	}

	decoded := decodeHuffman(asuint, metaR.Dict)

	return decoded, nil
}

type CompressedFileMetaData struct {
	EncodedLen  int                       `json:"e"`
	Dict        map[byte]CharPathEncoding `json:"d"`
	PaddingSize int                       `json:"ps"`
}

const META_SEPARATOR = "#"

func main() {
	filename := "data.txt"

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
