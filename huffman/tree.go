package huffman

import (
	"container/heap"
	"fmt"
	"slices"
	"stinky-compression/bwt"
	"stinky-compression/mft"
	proto_data "stinky-compression/proto/proto-data"
)

func printTree(node *Node, prefix string, isLeft bool, isFirst bool) {
	if node == nil {
		return
	}

	if isLeft && !isFirst {
		fmt.Print(prefix + "├─L-")
		prefix += "│ "
	} else if !isLeft && !isFirst {
		fmt.Print(prefix + "└─R-")
		prefix += " "
	}

	if node.IsAccNode {
		fmt.Printf("(%d)\n", node.Freq)
	} else {
		fmt.Printf("%d:%d\n", node.Char, node.Freq)
	}

	printTree(node.Left, prefix, true, false)
	printTree(node.Right, prefix, false, false)
}

func (n *Node) DebugTree() {
	printTree(n, "", false, true)
}

type path struct {
	path  uint64
	depth int
}

func (p *path) left() {
	p.path <<= 1
	p.depth += 1
}

func (p *path) right() {
	p.path <<= 1
	p.path |= 1
	p.depth += 1
}

func (p *path) up() {
	if p.depth > 0 {
		p.path >>= 1
		p.depth -= 1
	}
}

// to represent 0bxxx where x is the binary num
func treeToDict(node *Node, dict EncodingTable, path *path) {
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

	path.left()
	treeToDict(node.Left, dict, path)
	path.up()

	path.right()
	treeToDict(node.Right, dict, path)
	path.up()

}

type Node struct {
	Char      byte
	Freq      int
	Left      *Node
	Right     *Node
	IsAccNode bool
}

type charEncoding struct {
	Val  byte
	Freq int
}

type CharPathEncoding struct {
	Path uint64
	Size int
	Freq int
}

type FrequencyTable map[byte]int

func FrequencyTableToProto(table FrequencyTable) []*proto_data.CompressedFileMetaData_Frequency {
	protoTable := []*proto_data.CompressedFileMetaData_Frequency{}

	for key, val := range table {
		protoTable = append(protoTable, &proto_data.CompressedFileMetaData_Frequency{
			Char:      []byte{key},
			Frequency: int32(val),
		})
	}

	return protoTable
}

func ProtoFrequenciesToFrequencyTable(freqs []*proto_data.CompressedFileMetaData_Frequency) FrequencyTable {
	table := FrequencyTable{}

	for _, freq := range freqs {
		table[freq.GetChar()[0]] = int(freq.GetFrequency())
	}

	return table
}

type NodeHeap []*Node

func (h NodeHeap) Len() int { return len(h) }

func (h NodeHeap) Less(i, j int) bool {
	// freq comp
	if h[i].Freq != h[j].Freq {
		return h[i].Freq < h[j].Freq
	}

	// byte comp
	if (h[i].Left == nil && h[i].Right == nil) && (h[j].Left == nil && h[j].Right == nil) {
		return h[i].Char < h[j].Char
	}

	// idx comp
	return i < j
}

func (h NodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *NodeHeap) Push(node interface{}) {
	*h = append(*h, node.(*Node))
}

func (h *NodeHeap) Pop() interface{} {
	old := *h
	size := len(old)
	toPop := old[size-1]
	*h = old[0 : size-1]

	return toPop
}

func extractLenghts(node *Node, depth int, lenghts map[byte]int) {
	if node == nil {
		return
	}

	if node.Left == nil && node.Right == nil {
		lenghts[node.Char] = depth
		return
	}

	extractLenghts(node.Left, depth+1, lenghts)
	extractLenghts(node.Right, depth+1, lenghts)
}

func codeLengths(chars []charEncoding) map[byte]int {
	nodes := make(NodeHeap, 0, len(chars))

	for _, char := range chars {
		nodes = append(nodes, &Node{
			Char: char.Val,
			Freq: char.Freq,
		})
	}

	heap.Init(&nodes)

	for nodes.Len() > 1 {
		left := heap.Pop(&nodes).(*Node)
		right := heap.Pop(&nodes).(*Node)

		newNode := &Node{
			Freq:  left.Freq + right.Freq,
			Left:  left,
			Right: right,
		}

		heap.Push(&nodes, newNode)
	}

	lengths := map[byte]int{}
	if nodes.Len() > 0 {
		root := heap.Pop(&nodes).(*Node)
		extractLenghts(root, 0, lengths)
	}

	return lengths
}

type canonicalCodeSymbol struct {
	symbol byte
	length int
}

func genCanonicalCodes(lengths map[byte]int, freqT FrequencyTable) EncodingTable {
	symbols := make([]canonicalCodeSymbol, 0, len(lengths))

	for char, length := range lengths {
		symbols = append(symbols, canonicalCodeSymbol{
			symbol: char,
			length: length,
		})
	}

	slices.SortFunc(symbols, func(a, b canonicalCodeSymbol) int {
		if a.length != b.length {
			if a.length > b.length {
				return 1
			}

			return -1
		}

		if a.symbol > b.symbol {
			return 1
		}

		if a.symbol < b.symbol {
			return -1
		}

		return 0
	})

	res := map[byte]CharPathEncoding{}
	code := uint64(0)
	prevLen := 0

	for _, symb := range symbols {
		if symb.length > prevLen {
			code <<= uint(symb.length - prevLen)
			prevLen = symb.length
		}

		res[symb.symbol] = CharPathEncoding{
			Path: code,
			Size: symb.length,
			Freq: freqT[symb.symbol],
		}

		code++
	}

	return res
}

func buildCanonicalTree(codes map[byte]CharPathEncoding) *Node {
	root := &Node{IsAccNode: true}

	for sym, enc := range codes {
		node := root
		prevNode := root
		for pos := enc.Size - 1; pos >= 0; pos-- {
			bit := (enc.Path >> uint(pos)) & 1
			if bit == 1 {
				if node.Right == nil {
					node.Right = &Node{IsAccNode: true}
				}

				if node.Right != nil && node.Left != nil {
					node.Freq = node.Left.Freq + node.Right.Freq
				}

				prevNode = node
				node = node.Right
			} else {
				if bit == 0 && node.Left == nil {
					node.Left = &Node{IsAccNode: true}
				}

				if node.Right != nil && node.Left != nil {
					node.Freq = node.Left.Freq + node.Right.Freq
				}

				prevNode = node
				node = node.Left
			}
		}

		node.Char = sym
		node.Freq = enc.Freq
		node.IsAccNode = false
		if prevNode.Left != nil && prevNode.Right != nil {
			prevNode.Freq = prevNode.Left.Freq + prevNode.Right.Freq
		}
	}

	return root
}

func TreeFromFrequencies(input FrequencyTable) *Node {
	asList := []charEncoding{}

	for key, val := range input {
		asList = append(asList, charEncoding{
			Val:  key,
			Freq: val,
		})
	}

	slices.SortFunc(asList, func(a, b charEncoding) int {
		if a.Freq != b.Freq {
			if a.Freq > b.Freq {
				return 1
			}

			return -1
		}

		if a.Val > b.Val {
			return 1
		}

		if a.Val < b.Val {
			return -1
		}

		return 0
	})

	lengths := codeLengths(asList)
	codes := genCanonicalCodes(lengths, input)
	return buildCanonicalTree(codes)
}

func HuffmanEncoding(input []byte, debugMode bool) ([]CharPathEncoding, FrequencyTable, int) {
	bwtCoded, pIdx := bwt.Bwt(input)
	mftCoded := mft.Mft(bwtCoded)

	occurance := FrequencyTable{}
	for _, bt := range mftCoded {
		occurance[bt]++
	}

	asTree := TreeFromFrequencies(occurance)
	if debugMode {
		asTree.DebugTree()
	}

	charDict := EncodingTable{}
	treeToDict(asTree, charDict, &path{})

	encoded := []CharPathEncoding{}
	for _, bt := range mftCoded {
		encoded = append(encoded, charDict[bt])
	}

	return encoded, occurance, pIdx
}

func DecodeCompressionFromTable(bits []CharPathEncoding, dict FrequencyTable, bwtIdx int) []byte {
	tree := TreeFromFrequencies(dict)
	decoded := []byte{}
	for _, bit := range bits {
		head := tree
		for pos := bit.Size - 1; pos >= 0; pos-- {
			b := (bit.Path >> uint(pos)) & 1
			if b == 1 {
				head = head.Right
			} else {
				head = head.Left
			}
		}

		decoded = append(decoded, head.Char)
	}

	mftDecodd := mft.DecodeMft(decoded)
	bwtDecoded := bwt.DecodeBwt(mftDecodd, bwtIdx)

	return bwtDecoded
}

type EncodingTable map[byte]CharPathEncoding
