package tree

import (
	"fmt"
	"sort"
	"strings"
)

type SymbolFrequencies []int

func MakeSymbolFrequencies() SymbolFrequencies {
	sf := SymbolFrequencies(make([]int, 256))

	return sf
}

func (sf SymbolFrequencies) RegisterOccurrence(b byte) {
	sf[int(b)]++
}

func (sf SymbolFrequencies) String() string {
	result := "byte\tsymbol\tcount\n"
	for i, count := range sf {
		if count == 0 {
			continue
		}
		result += fmt.Sprintf("% 00x\t%s\t%d\n", byte(i), string([]byte{byte(i)}), count)
	}
	return result
}

func CountSymbols(text []byte) SymbolFrequencies {
	frequencies := MakeSymbolFrequencies()
	for _, symbol := range text {
		frequencies.RegisterOccurrence(symbol)
	}

	return frequencies
}

type WeightedSymbols struct {
	Symbols []byte
	Weight  int
	Node    *Node
}

func (w WeightedSymbols) String() string {
	return fmt.Sprintf("<0x% 00x | %s> %d", w.Symbols, string(w.Symbols), w.Weight)
}

type Node struct {
	Parent   *Node
	Children [2]*Node
	Symbols  WeightedSymbols
}

func (n *Node) String() string {
	return strings.Join(
		[]string{
			"NODE:",
			"symbols:",
			"\t" + fmt.Sprint(n.Symbols),
			"children: ",
			"\t" + fmt.Sprint(n.Children[0]),
			"\t" + fmt.Sprint(n.Children[1]),
		},
		"\n",
	)
}

func (n *Node) PadCode() (paddedCode byte, codeLen int) {
	code := n.Code()
	padded := make([]int, 8)
	codeLen = copy(padded, code)
	var codeByte byte = 0b00000000
	for dstPos := 7; dstPos >= 0; dstPos-- {
		setBit := byte(padded[7-dstPos]) << dstPos
		codeByte = codeByte | setBit
	}

	return codeByte, codeLen
}

func (n *Node) Concat(m *Node) *Node {
	newNode := &Node{Children: [2]*Node{n, m}}
	n.Parent = newNode
	m.Parent = newNode

	newSymbols := WeightedSymbols{
		Symbols: append(n.Symbols.Symbols, m.Symbols.Symbols...),
		Weight:  n.Symbols.Weight + m.Symbols.Weight,
		Node:    newNode,
	}
	newNode.Symbols = newSymbols

	return newNode
}

func (n *Node) IsLeaf() bool {
	return n.Children[0] == nil && n.Children[1] == nil
}

const Unidentified int = -1

func (n *Node) Identify(child *Node) int {
	for id, c := range n.Children {
		if c == child {
			return id
		}
	}

	return Unidentified
}

func (n *Node) Ancestry() (parent *Node, codon int) {
	if n.Parent == nil {
		return nil, Unidentified
	}

	parent, codon = n.Parent, n.Parent.Identify(n)
	return parent, codon
}

func (n *Node) Code() (code []int) {
	if n.Parent == nil {
		return []int{}
	}
	next := n.Ancestry
	for parent, codon := next(); parent != nil; parent, codon = next() {
		next = parent.Ancestry
		code = append([]int{codon}, code...)
	}

	return code
}

func (n *Node) Walk(callback func(n *Node, c []int), code []int) {
	callback(n, code)
	if n.IsLeaf() {
		return
	}
	for idx, child := range n.Children {
		childCode := append(code, idx)
		if child != nil {
			child.Walk(callback, childCode)
		}
	}
}

type NodeArray []*Node

func (na NodeArray) Len() int {
	return len(na)
}

func (na NodeArray) Less(i, j int) bool {
	return na[i].Symbols.Weight < na[j].Symbols.Weight
}

func (na NodeArray) Swap(i, j int) {
	na[i], na[j] = na[j], na[i]
}

func FromFrequencies(sf SymbolFrequencies) NodeArray {
	na := []*Node{}
	for i, v := range sf {
		// bytes that don't appear are ignored
		if v == 0 {
			continue
		}
		node := &Node{
			Symbols: WeightedSymbols{Symbols: []byte{byte(i)}, Weight: v},
		}
		node.Symbols.Node = node
		na = append(na, node)
	}

	return NodeArray(na).Sort()
}

func (na NodeArray) Include(n *Node) NodeArray {
	na = NodeArray(append(na, n))

	return na.Sort()
}

func (na NodeArray) Sort() NodeArray {
	// slice being a pointer in disguise is a pain
	// this code prevents this method modifying na
	// by creating a new slice and having the sort
	// in place operate on that
	sorted := make(NodeArray, len(na))
	_ = copy(sorted, na)
	sort.Sort(&sorted)
	return sorted
}

func (na NodeArray) String() string {
	elems := make([]string, na.Len())
	for i, node := range na {
		elems[i] = "<" + string(node.Symbols.Symbols) + ">"
	}

	return strings.Join(elems, ";")
}

func (na NodeArray) PopNextPair() ([2]*Node, NodeArray) {
	na.Sort()
	pair := make([]*Node, 2)
	pair = na[0:2]
	na = NodeArray(na[2:])
	return [2]*Node{pair[0], pair[1]}, na.Sort()
}
