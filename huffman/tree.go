package huffman

import (
	"fmt"
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
	Char  byte
	Freq  int
	Left  *Node
	Right *Node
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

func (c *CharPathEncoding) makeSafePathMeta() {
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
func (c *CharPathEncoding) decodeSafePathMetaToPath() {
	var bin uint64
	for _, char := range c.PathMeta {
		bin <<= 1 // shift by 1
		if char == '1' {
			bin |= 1 // flip 0 to 1
		}
	}

	c.Path = bin
}

func TreeFromEncodingTable(dict EncodingTable) *Node {
	nodes := []Node{}
	for char, enc := range dict {
		nodes = append(nodes, Node{
			Char: char,
			Freq: enc.Freq,
		})
	}

	slices.SortFunc(nodes, func(a, b Node) int {
		aEnc := dict[a.Char]
		bEnc := dict[b.Char]

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
		aEnc := dict[a.Char]
		mask := uint64(3)
		aLastBts := aEnc.Path & mask

		bEnc := dict[b.Char]
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
		lastBit := (dict[node.Char].Path >> uint(0)) & 1
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

func encodeToTree(chars []charEncoding) *Node {
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

func HuffmanEncoding(input []byte, debugMode bool) ([]CharPathEncoding, EncodingTable) {
	occurance := map[byte]int{}
	for _, bt := range input {
		if _, ok := occurance[bt]; ok {
			occurance[bt] += 1
		} else {
			occurance[bt] = 1
		}
	}

	asList := []charEncoding{}
	for key, val := range occurance {
		asList = append(asList, charEncoding{
			Val:  key,
			Freq: val,
		})
	}

	slices.SortFunc(asList, func(a, b charEncoding) int {
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
		asTree.DebugTree()
	}

	charDict := EncodingTable{}
	treeToDict(asTree, charDict, &path{})

	encoded := []CharPathEncoding{}
	for _, bt := range input {
		encoded = append(encoded, charDict[bt])
	}

	return encoded, charDict
}

func DecodeCompressionFromTable(bits []CharPathEncoding, dict EncodingTable) string {
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

type EncodingTable map[byte]CharPathEncoding

func (et *EncodingTable) MakeSafePathMetaForMetadata() {
	dict := *et

	for key, val := range dict {
		val.makeSafePathMeta()
		dict[key] = val
	}

	et = &dict
}

func (et *EncodingTable) DecodeSafePathMetaFromTable() {
	dict := *et

	for key, val := range dict {
		val.decodeSafePathMetaToPath()
		dict[key] = val
	}

	et = &dict
}
