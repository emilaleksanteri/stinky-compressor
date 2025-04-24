package huffman

import (
	"container/heap"
	"fmt"
	"slices"
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
	Path     uint64 `json:"-"`
	PathMeta string `json:"p"`
	Size     int    `json:"s"`
	Freq     int    `json:"f"`
}

type FrequencyTable map[byte]int

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

func genCanonicalCodes(lengths map[byte]int) map[byte]CharPathEncoding {
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
			Freq: lengths[symb.symbol],
		}

		code++
	}

	return res
}

func buildCanonicalTree(codes map[byte]CharPathEncoding) *Node {
	root := &Node{IsAccNode: true}

	for sym, enc := range codes {
		node := root
		for pos := enc.Size - 1; pos >= 0; pos-- {
			bit := (enc.Path >> uint(pos)) & 1
			if bit == 1 {
				if node.Right == nil {
					node.Right = &Node{IsAccNode: true}
				}
				node = node.Right
			} else {
				if bit == 0 && node.Left == nil {
					node.Left = &Node{IsAccNode: true}
				}
				node = node.Left
			}
		}

		node.Char = sym
		node.IsAccNode = false
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
	codes := genCanonicalCodes(lengths)
	return buildCanonicalTree(codes)
}

func HuffmanEncoding(input []byte, debugMode bool) ([]CharPathEncoding, FrequencyTable) {
	occurance := FrequencyTable{}
	for _, bt := range input {
		if _, ok := occurance[bt]; ok {
			occurance[bt] += 1
		} else {
			occurance[bt] = 1
		}
	}

	asTree := TreeFromFrequencies(occurance)
	if debugMode {
		asTree.DebugTree()
	}

	charDict := EncodingTable{}
	treeToDict(asTree, charDict, &path{})

	encoded := []CharPathEncoding{}
	for _, bt := range input {
		encoded = append(encoded, charDict[bt])
	}

	return encoded, occurance
}

func DecodeCompressionFromTable(bits []CharPathEncoding, dict FrequencyTable) []byte {
	tree := TreeFromFrequencies(dict)
	charDict := EncodingTable{}
	treeToDict(tree, charDict, &path{})

	decoded := []byte{}
	for _, bit := range bits {
		for char, enc := range charDict {
			if enc.Path == bit.Path {
				decoded = append(decoded, char)
				break
			}
		}
	}

	return decoded
}

type EncodingTable map[byte]CharPathEncoding
