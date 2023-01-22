package encoder

import "testing"

var source = []byte("In computer science and information theory, a Huffman code is a particular type of optimal prefix code that is commonly used for lossless data compression.\n")

func TestEncoder_Encode(t *testing.T) {
	encoder := EncoderFromData(source)
	testCases := []struct {
		in  []byte
		out []byte
	}{
		// the coded message always expresses the number of unoccupied bits in the penultimate
		// byte with the final byte, hence the decimal expression of the final byte in each output value
		{[]byte{0x49, 0x6f}, []byte{0b01011001, 0b11100000, 5}},
		{[]byte{'c', 'o', 'd', 'e'}, []byte{0b00101111, 0b10010101, 0b00000000, 7}},
	}

	for _, testCase := range testCases {
		actualCoded := encoder.Encode(testCase.in)
		if len(actualCoded) > len(testCase.out) {
			t.Errorf(
				"The expected length of the coded bytes was %d, got %d",
				len(testCase.out), len(actualCoded),
			)
			t.FailNow()
		}

		for position, actualCodedByte := range actualCoded {
			if testCase.out[position] != actualCodedByte {
				t.Errorf(
					"Input was %s, expected byte 0b%08b but got 0b%08b at position %d",
					string(testCase.in), testCase.out[position], actualCodedByte, position,
				)
			}
		}
	}
}

func TestEncoder_Decode(t *testing.T) {
	encoder := EncoderFromData(source)
	testCases := []struct {
		in  []byte
		out []byte
	}{
		// the coded message always expresses the number of unoccupied bits in the penultimate
		// byte with the final byte, hence the decimal expression of the final byte in each input value
		{[]byte{0b01011001, 0b11100000, 5}, []byte{0x49, 0x6f}},
		{[]byte{0b00101111, 0b10010101, 0b00000000, 7}, []byte{'c', 'o', 'd', 'e'}},
	}

	for i, testCase := range testCases {
		actualMessage := encoder.Decode(testCase.in)
		if len(actualMessage) > len(testCase.out) {
			t.Errorf(
				"Case %d: The expected length of the message bytes was %d, got %d\nexpected: % x\nactual: % x",
				i, len(testCase.out), len(actualMessage), testCase.out, actualMessage,
			)
			t.FailNow()
		}

		for position, actualSymbol := range actualMessage {
			if testCase.out[position] != actualSymbol {
				t.Errorf(
					"Input was %s, expected symbol 0b%c but got 0b%c at position %d",
					string(testCase.in), testCase.out[position], actualSymbol, position,
				)
			}
		}

	}
}

func TestEncoder_EndToEnd(t *testing.T) {
	encoder := EncoderFromData(source)
	encoded := encoder.EncodeAllWithTable()

	decoder := EncoderFromEncoded(encoded)
	decoded := decoder.Message()

	if string(source) != string(decoded) {
		t.Errorf("source data: \n%s\n\nafter encode then decode: \n%s\n", string(source), string(encoded))
	}
}
