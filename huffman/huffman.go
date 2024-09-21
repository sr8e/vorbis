package huffman

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type HuffmanTree struct {
	root    *Node
	current *Node
}

type Node struct {
	index  int
	parent *Node
	left   *Node
	right  *Node
}

// isAvailable returns if you can append a child to the node.
func (n *Node) isAvailable() bool {
	return n.index == -1 && n.right == nil
}

func (n *Node) appendChild(child *Node) {
	if !n.isAvailable() {
		return
	}
	child.parent = n
	if n.left != nil {
		n.right = child
		return
	}
	n.left = child
}

func (n *Node) isValid() bool {
	if n.index != -1 {
		return n.left == nil && n.right == nil
	} else {
		return n.left != nil && n.right != nil && n.left.isValid() && n.right.isValid()
	}
}

func (n *Node) iterateCodeword(cw string) string {
	if n.index == -1 {
		return fmt.Sprintf("%s\n%s", n.left.iterateCodeword(cw+"0"), n.right.iterateCodeword(cw+"1"))
	} else {
		return fmt.Sprintf("entry %d, codeword %s", n.index, cw)
	}
}

func (n *Node) iterateStructure() string {
	filler := "\n     "
	if n.index == -1 {
		var strL, strR []string
		if n.left != nil {
			strL = strings.Split(n.left.iterateStructure(), "\n")
		}
		if n.right != nil {
			strR = strings.Split(n.right.iterateStructure(), "\n")
		}
		if strL == nil {
			return "[  ]"
		} else if strR == nil {
			return fmt.Sprintf("[  ]───%s", strings.Join(strL, filler+"  "))
		} else {
			return fmt.Sprintf("[  ]─┬─%s%s└─%s", strings.Join(strL, filler+"│ "), filler, strings.Join(strR, filler+"  "))
		}
	} else {
		return fmt.Sprintf("[%2d]", n.index)
	}
}

func GenerateHuffmanTree(cwLen []int) (_ HuffmanTree, err error) {

	root := &Node{index: -1}
	cwMaxLen := slices.Max(cwLen)
	minNode := make([]*Node, cwMaxLen+1)
	minNode[0] = root

	for i := 1; i <= cwMaxLen; i++ {
		minNode[i] = &Node{index: -1}
		minNode[i-1].appendChild(minNode[i])
	}
	minNode[0] = nil

	for i, cl := range cwLen {
		if cl == -1 { // sparse tree, unused entry
			continue
		}

		leaf := minNode[cl]
		if leaf == nil {
			err = errors.New("the tree is overpopulated")
			return
		}
		leaf.index = i
		if leaf.left != nil {
			leaf.left = nil
		}

		for j := cl; j > 0; j-- { // search toward root
			newNode := &Node{index: -1}
			if pNode := minNode[j].parent; pNode.isAvailable() { // increment codeword
				pNode.appendChild(newNode)
				minNode[j] = newNode
			} else if uNode := minNode[j-1]; uNode != nil && uNode.isAvailable() { // jump branch
				uNode.appendChild(newNode)
				minNode[j] = newNode
				break
			} else { // no codeword of length j anymore
				minNode[j] = nil
				break
			}
		}

		if cl < cwMaxLen && minNode[cl+1].parent == leaf && minNode[cl] != nil && minNode[cl].isAvailable() {
			minNode[cl].appendChild(minNode[cl+1])
		}
	}

	// now all nodes should not be available
	if !root.isValid() {
		err = errors.New("the tree is underpopulated")
		return
	}

	return HuffmanTree{root: root}, nil
}

func (ht *HuffmanTree) String() string {
	return ht.root.iterateStructure() + "\n" + ht.root.iterateCodeword("")
}
