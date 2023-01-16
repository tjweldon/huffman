package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
)

type Cli struct {
	Source string `arg:"--source" help:"The text file to compress" default:"example.txt"`
}

var args Cli

func init() {
	arg.MustParse(&args)
}

func loadText(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

type SymbolFrequencies map[byte]int

func (sf SymbolFrequencies) RegisterOccurrence(b byte) {
	count, repeated := sf[b]
	if !repeated {
		count = 0
	}

	sf[b] = count + 1
}

func (sf SymbolFrequencies) String() string {
	result := "byte\tsymbol\tcount\n"
	for k, v := range sf {
		result += fmt.Sprintf("% 00x\t%s\t%d\n", k, string([]byte{k}), v)
	}
	return result
}

func CountSymbols(text []byte) SymbolFrequencies {
	frequencies := SymbolFrequencies(make(map[byte]int))
	for _, symbol := range text {
		frequencies.RegisterOccurrence(symbol)
	}

	return frequencies
}

type WeightedSymbols struct {
	Symbols []byte
	Weight int
	Node *Node
}

func (w WeightedSymbols) String() string {
	return fmt.Sprintf("<0x% 00x | %s> %d", w.Symbols ,string(w.Symbols), w.Weight)
}

type Node struct {
	Parent *Node
	Children [2]*Node
	Symbols WeightedSymbols
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

func (n *Node) Concat(m *Node) *Node {
	newNode := &Node{Children: [2]*Node{n, m}}
	n.Parent = newNode
	m.Parent = newNode

	newSymbols := WeightedSymbols{
		Symbols: append(n.Symbols.Symbols, m.Symbols.Symbols...),
		Weight: n.Symbols.Weight + m.Symbols.Weight,
		Node: newNode,
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
	for k, v := range sf {
		node := &Node{
			Symbols: WeightedSymbols{Symbols: []byte{k}, Weight: v},
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
	sort.Sort(&na)
	return na
}

func (na NodeArray) String() string {
	elems := make([]string, na.Len())
	for i, node := range na {
		elems[i] = "<"+string(node.Symbols.Symbols)+">"
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

func main() {
	text := loadText(args.Source)
	freqs := CountSymbols(text)
	na := FromFrequencies(freqs)
	na = na.Sort()

	var pair [2]*Node
	for na.Len() > 2 {
		pair, na = na.PopNextPair()
		fmt.Printf("pair <%s>, <%s>\n", pair[0].Symbols.Symbols, pair[1].Symbols.Symbols)
		fmt.Printf("na %s\n", na)
		subtree := pair[0].Concat(pair[1])
		na = na.Include(subtree)	
	}
	
	tree := na[0].Concat(na[1])
	
	callback := func(n *Node, code []int) {
		//fmt.Printf("CALLBACK: %v\n", n)
		if !n.IsLeaf() {
			return
		} else {
			replaced := strings.Replace(string(n.Symbols.Symbols), "\n", "\\n", -1)
			replaced = strings.Replace(replaced, " ", "\\s", -1)
			
			tableRow := []string{
				fmt.Sprintf("%#02x", n.Symbols.Symbols[0]),
				replaced,
				fmt.Sprint(n.Symbols.Weight),
				fmt.Sprintf("%-9s", fmt.Sprint(code)),
				
				fmt.Sprint(len(code)*n.Symbols.Weight),
				fmt.Sprint(n.Symbols.Weight * 8),
				fmt.Sprint(float64(len(code)*n.Symbols.Weight)/float64(8*n.Symbols.Weight)),
				fmt.Sprint(n.Code()),
			}
			fmt.Println(strings.Join(tableRow, "\t"))
		}
	}
	fmt.Println(strings.Join([]string{
		"hex", "ascii", "freq", "huffman\t", "smol", "normal", "ratio", "n.Code()",
	}, "\t"))
	
	tree.Walk(callback, []int{})

	fmt.Println("tree.Code()", tree.Code())
	fmt.Println("tree.Children[0].Code()", tree.Children[0].Children[1].Code())
}
