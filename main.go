package main

import (
	"log"
	"os"
	"strings"

	enc "tjweldon/huffman/src/encoder"

	"github.com/alexflint/go-arg"
)

type Cli struct {
	Encode *EncodeMode `arg:"subcommand:encode" help:"use the --help flag with this subcommand to see its docs"`	
	Decode *DecodeMode `arg:"subcommand:decode" help:"use the --help flag with this subcommand to see its docs"`	
}

type EncodeMode struct {
	Source string `arg:"positional" help:"The text file to compress"`
	Destination string `arg:"-o, --out" help:"Where to write the output"`
}

func (em *EncodeMode) Run() {
	if em.Source == "" {
		em.Source = "./example.txt"
	}
	if em.Destination == "" {
		fragments := strings.Split(em.Source, ".")
		dstFragments := append(fragments[:len(fragments) - 1], "huff")
		em.Destination = strings.Join(dstFragments, ".")
	}

	encoder := enc.EncoderFromData(loadFile(em.Source))
	encoded := encoder.EncodeAllWithTable()
	writeFile(em.Destination, encoded)
}

type DecodeMode struct {
	Source string `arg:"positional" help:"The text file to compress"`
	Destination string `arg:"-o, --out" help:"Where to write the output"`
}

func (dm *DecodeMode) Run() {
	if dm.Source == "" {
		log.Fatal("No source file supplied to decode")
	}
	if dm.Destination == "" {
		fragments := strings.Split(dm.Source, ".")
		dstFragments := append(fragments[:len(fragments) - 1], "txt")
		dm.Destination = strings.Join(dstFragments, ".")
	}

	encoder := enc.EncoderFromEncoded(loadFile(dm.Source))
	decoded := encoder.Message()
	writeFile(dm.Destination, decoded)
}

var args Cli

func init() {
	arg.MustParse(&args)
}

func loadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func writeFile(path string, data []byte) {
	if err := os.WriteFile(path, data, os.FileMode(0o644)); err != nil {
		log.Fatal(err)
	}
}

func main() {
	switch {
	case args.Encode != nil:
		args.Encode.Run()
	case args.Decode != nil:
		args.Decode.Run()
	}
}


