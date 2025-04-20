package main

import (
	"fmt"
	"slices"
)

func printTree(node *Node, prefix string, isLeft bool) {
	if node == nil {
		return
	}

	if !isLeft {
		fmt.Print(prefix + "└─")
		prefix += "  "
	} else {
		fmt.Print(prefix + "├─")
		prefix += "│ "
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
func treeToDict(node *Node, dict map[byte]uint64, path *Path) {
	if node == nil {
		return
	}

	if node.Left == nil && node.Right == nil {
		dict[node.Char] = path.path
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
	// make into pair
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

func main() {
	encodeStr := "hello world!"
	asBts := []byte(encodeStr)

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
	charDict := map[byte]uint64{}
	treeToDict(asTree, charDict, &Path{})

	encoded := []uint64{}
	for _, bt := range asBts {
		encoded = append(encoded, charDict[bt])
	}

	decoded := ""
	for _, u := range encoded {
		for key, val := range charDict {
			if val == u {
				decoded += string(key)
				break
			}
		}
	}

	fmt.Printf("decoded: %s\n", decoded)
	fmt.Println(len(decoded))

}
