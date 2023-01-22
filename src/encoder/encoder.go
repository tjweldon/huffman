package encoder

import (
	"fmt"
	"log"
	"sort"
	"strings"

	t "tjweldon/huffman/src/tree"
)

func Sanitise(s []byte) string {
	replaced := strings.Replace(string(s), "\n", "\\n", -1)
	replaced = strings.Replace(replaced, " ", "\\s", -1)
	return replaced
}

type HuffmanCode struct {
	Symbol  byte
	Code    byte
	CodeLen byte
}

func UnpackCode(code [3]byte) HuffmanCode {
	return HuffmanCode{
		Symbol:  code[0],
		Code:    code[1],
		CodeLen: code[2],
	}
}

func (hc HuffmanCode) String() string {
	return fmt.Sprintf(
		"symbol: <0x%02x|% -2s>, padded code: %08b, significant bits: %d",
		hc.Symbol, Sanitise([]byte{hc.Symbol}), hc.Code, hc.CodeLen,
	)
}

func (hc HuffmanCode) CopyInto(buf []byte) {
	if n := copy(buf, []byte{hc.Symbol, hc.Code, hc.CodeLen}); n != 3 {
		log.Fatal("could not copy code. hc=", hc, " n=", n)
	}
}

type CodeTable struct {
	codes   []HuffmanCode
	indices *struct {
		symbol2code [][2]byte
		code2symbol []byte
	}
}

func DecodeBytes(rawTable []byte) *CodeTable {
	remaining := rawTable
	codes := []HuffmanCode{}
	for len(remaining) >= 3 {
		triplet := [3]byte{}
		if n := copy(triplet[:], remaining); n != 3 {
			log.Fatal("huffman code had encoded length != 3. The actual length is", n)
		}
		remaining = remaining[3:]
		codes = append(codes, UnpackCode(triplet))
	}

	ct := &CodeTable{codes: codes}
	ct.populateIndices()
	return ct
}

func PopTable(compressedData []byte) (ct *CodeTable, encodedData []byte) {
	alphabetSize, encodedData := compressedData[0], compressedData[1:]
	rawTable, encodedData := encodedData[:alphabetSize*3], encodedData[alphabetSize*3:]
	ct = DecodeBytes(rawTable)
	if ct.Len() != int(alphabetSize) {
		log.Fatal(
			"Alphabet size != codeTable.Len(). ",
			"alphabetSize=", alphabetSize,
			" ct.Len()=", ct.Len(),
		)
	}

	return ct, encodedData
}

func FromNodes(index t.NodeArray) *CodeTable {
	table := &CodeTable{codes: []HuffmanCode{}}
	for _, node := range index.Sort() {
		padded, codeLen := node.PadCode()
		table.codes = append(
			table.codes,
			HuffmanCode{
				Symbol:  node.Symbols.Symbols[0],
				Code:    padded,
				CodeLen: byte(codeLen),
			},
		)
	}

	table.populateIndices()

	return table
}

func (ct *CodeTable) populateIndices() {
	indices := struct {
		symbol2code [][2]byte
		code2symbol []byte
	}{
		make([][2]byte, 256),
		make([]byte, 256),
	}
	for _, hc := range ct.codes {
		indices.symbol2code[hc.Symbol] = [2]byte{hc.Code, hc.CodeLen}
		indices.code2symbol[hc.Code] = hc.Symbol
	}
	ct.indices = &indices
	sort.Sort(ct)
}

func (ct *CodeTable) Less(i, j int) bool {
	return ct.codes[i].CodeLen < ct.codes[j].CodeLen
}

func (ct *CodeTable) Len() int {
	return len(ct.codes)
}

func (ct *CodeTable) Swap(i, j int) {
	ct.codes[i], ct.codes[j] = ct.codes[j], ct.codes[i]
}

func (ct *CodeTable) EncodeBytes() []byte {
	result := make([]byte, len(ct.codes)*3)
	for i, symbolCode := range ct.codes {
		start := 3 * i
		stop := start + 3
		symbolCode.CopyInto(result[start:stop])
	}

	return result
}

func (ct *CodeTable) EncodeHeader() byte {
	return byte(len(ct.codes))
}

func (ct *CodeTable) Bytes() []byte {
	return append([]byte{ct.EncodeHeader()}, ct.EncodeBytes()...)
}

func (ct *CodeTable) String() string {
	rows := []string{}
	for _, code := range ct.codes {
		rows = append(rows, fmt.Sprint(code))
	}
	rows = append(rows, "")

	return strings.Join(rows, "\n")
}

func (ct *CodeTable) GetCode(symbol byte) (code byte, length int) {
	if ct.indices == nil {
		ct.populateIndices()
	}
	cl := ct.indices.symbol2code[symbol]
	return cl[0], int(cl[1])
}

func (ct *CodeTable) GetSymbol(paddedCode byte) byte {
	if ct.indices == nil {
		ct.populateIndices()
	}
	return ct.indices.code2symbol[paddedCode]
}

const NoMatch int = -1

//	v     v       v
//
// 0 1 1 0 1 1 1 1|1 1
func (ct *CodeTable) PopDecode(buf uint16) (symbol byte, shift int) {
	shift = NoMatch
	mostSignificantByte := byte(buf >> 8)
	sort.Sort(ct)
	for _, hc := range ct.codes {
		testShift := 8 - hc.CodeLen

		match := (mostSignificantByte - hc.Code) >> testShift
		if match == 0 {
			symbol, shift = hc.Symbol, int(hc.CodeLen)
			break
		}
	}
	return symbol, shift
}

const UnknownSymbol int = -1

func (ct *CodeTable) InsertCode(buf uint16, at int, symbol byte) (result uint16, shift int) {
	c, length := ct.GetCode(symbol)
	if length == 0 {
		log.Fatal("")
	}
	if at < length {
		return buf, 0
	}

	bufMask := uint16(uint16(c) << (at - 7))
	roundedBuf := (buf >> at) << at
	result = bufMask | roundedBuf
	return result, length
}

func (ct *CodeTable) CodeLenBounds() (shortest, longest int) {
	sort.Sort(ct)
	return int(ct.codes[0].CodeLen), int(ct.codes[ct.Len()-1].CodeLen)
}

type Encoder struct {
	ct   *CodeTable
	data []byte
}

func EncoderFromData(data []byte) *Encoder {
	e := &Encoder{}
	e.buildCodeTable(data)

	return e
}

func (e *Encoder) buildCodeTable(data []byte) {
	freqs := t.CountSymbols(data)
	indexNa := t.FromFrequencies(freqs)
	na := indexNa.Sort()

	var pair [2]*t.Node
	for na.Len() > 2 {
		pair, na = na.PopNextPair()
		subtree := pair[0].Concat(pair[1])
		na = na.Include(subtree).Sort()
	}

	na[0].Concat(na[1])
	e.ct = FromNodes(indexNa.Sort())
	e.data = data
}

func EncoderFromEncoded(encoded []byte) *Encoder {
	table, encMessage := PopTable(encoded)
	e := Encoder{ct: table}
	decoded := e.Decode(encMessage)
	e.data = decoded

	return &e
}

func (e *Encoder) Encode(data []byte) []byte {
	// initialisation of bit occupancy count, output slice and output buffer
	shift := 0
	result := []byte{}
	outputBuffer := uint16(0b00000000_00000000)

	for _, symbol := range data {
		var nextShift int

		// the symbol is encoded and written to the next available bits in the buffer
		outputBuffer, nextShift = e.ct.InsertCode(outputBuffer, 15-shift, symbol)

		// the new position of the last occupied bit is tracked
		shift += nextShift

		// when the length of encoded data in the buffer exceeds 8 bits,
		if shift > 7 {
			// the leftmost byte is appended to the output
			result = append(result, byte(outputBuffer>>8))

			// the buffer is then shifted 8 bits to the left.
			shift -= 8
			outputBuffer <<= 8
		}
	}

	// at this point there are only occupied bits in the leftmost byte
	// if not then something has gone horribly wrong
	if shift > 7 {
		log.Fatal("Something went wrong at the end of the file")
	}
	// if there are occupied bits in the buffer then append them to the data
	if shift > 0 {
		result = append(
			result,
			// the final occupied bits and the rest is padding
			byte(outputBuffer>>8),
			// the final bit is an unsigned int representing
			// the amount of padding in the penultimate bit
			byte(8-shift),
		)
	}

	return result
}

func (e *Encoder) Decode(data []byte) []byte {
	// initialisation of the outputBuffer, the number of unoccupied bits, and the output data slice
	outputBuffer := uint16(0b00000000_00000000)
	firstBytes, data := data[0:1], data[1:]
	outputBuffer += uint16(firstBytes[0]) << 8
	unoccupiedBits := 8
	decoded := []byte{}

	// the last byte is the number of unoccupied bits in the final byte of encoded data
	data, padding := data[:len(data)-1], data[len(data)-1]

	// sort by asc code length, record the length of the shortest code
	sort.Sort(e.ct)

	for {
		var encodedByte byte

		if unoccupiedBits <= 7 || len(data) == 0 {
			// when we are guaranteed not to exhaust the buffer with a read
			// or we have reached the end of the data and we know how many bits are left
			// pop a symbol from the buffer

			// append the matching symbol to the output and record the number of bits to discard
			symbol, codeLen := e.ct.PopDecode(outputBuffer)
			if codeLen == NoMatch {
				break
			}

			// discard the matched code by shifting it out of the buffer to the left
			outputBuffer <<= codeLen
			// after discarding the matched code there are codeLen more unoccupied bits
			unoccupiedBits += codeLen

			if unoccupiedBits > 16 {
				// if we read beyond the padding then just discard any match and we're done
				break
			}

			// otherwise that's good data baby
			decoded = append(decoded, symbol)

			if unoccupiedBits == 16 {
				// if there's only padding left at this point we're done
				break
			}

		} else if len(data) > 0 && unoccupiedBits >= 8 {
			// when the tail position has moved 8 bits leftward, and data remains to be decoded
			// pop a byte of input data, append it after the tail position

			// pop a byte from the data
			encodedByte, data = data[0], data[1:]

			// the shift pads right with zeros but leaves space for 8 bits so
			// the embedding ends 8 bits after the last code bit in the buffer
			shift := unoccupiedBits - 8
			embeddedCode := uint16(encodedByte) << shift

			// set the bits in the buffer so that the embedding is appended to the tail
			outputBuffer |= embeddedCode
			// after inserting a byte, there are 8 fewer unoccupied bits in the buffer
			unoccupiedBits -= 8

			// if we're out of data, then we need to account for the padding in the final byte
			// and mark the padding as unoccupied so we don't try to decode the padding
			if len(data) == 0 {
				unoccupiedBits += int(padding)
			}
		}
	}

	return decoded
}

func (e *Encoder) EncodeAll() []byte {
	return e.Encode(e.data)
}

func (e *Encoder) EncodeAllWithTable() []byte {
	return append(e.ct.Bytes(), e.EncodeAll()...)
}

func (e *Encoder) Table() *CodeTable {
	return e.ct
}

func (e *Encoder) Message() []byte {
	return e.data
}
