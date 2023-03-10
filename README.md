# Huffman

A project I put together using go to teach myself about the compression algorithm Huffman coding.
It presents a CLI to compress/decompress a source file and write the result to a destination file.

### Disclaimer

Bit manipulation and compression algorithms are not remotely within my wheelhouse,
so the code is probably overcomplicated and likely does not perform well against
large file sizes.

This was an intentional trade-off since attempting to learn about the technique and writing
an optimised and concise implementation of it would have been an unrealistic goal.

### What does it actually do?

For reference I used [this section](https://en.wikipedia.org/wiki/Huffman_coding#Basic_technique) of the Wikipedia article on Huffman coding
as the basis for my implementation, however that just gives me the bidirectional mapping from huffman code to input byte. This is implemented in the src/tree package.

I then had to go well outside my comfort zone and work out how to concatenate variable bit-length codes without any padding when creating the encoded message.
Since I wanted to be able to check that the implementation worked, I also needed to be able to consume the resulting data. This was by far the hardest part
but was super easy to unit test, though I didn't start test first. Because of the difficulty and the fact that I wanted to do all of the implementation myself,
the code for this is probably the messiest and most verbose. On the other hand it's also commented up for readability and unit tested so swings and roundabouts I suppose.

The encoded format looks something like this:

The format is `|semantics; encoding; multiplicity|` where the `|` represents the boundary between successive contiguous regions of the data e.g.
`|literally 1; u8; 128|` means literally the number 1 as an unsigned 8-bit integer, repeated 128 times.

The encoding `huffman` means the variable (max 8) bit length huffman coding of a symbol.

```
// The number of rows to parse when extracting the encoding table.
|code table length; u8; 1|											

// Each code stored as one u8 for the input symbol, the most significant bits of the next byte are the huffman code
// then the next byte expresses how much of the previous byte is to be ignored when decoding (i.e. 8 - <length of huffman code>).
|symbol, padded huffman code, padding size; 3*u8; code table length|	

// The encoded message does not necessarily end at a byte boundary
|encoded symbol; huffman; input length in bytes|  					

// Any unoccupied bits in the last byte are junk,
|0 bit literal; 1 bit; padding length| 								

//unfortunately we need to record how much of the junk to ignore on decode since one or more 0 bits is a valid huffman code.
|padding length; u8; 1|												
```

### Limitations

 - Only works on 8 bit encoded text (ascii etc.)
 - Does everything in memory so big input files may break it, or make it profoundly slow. I haven't tried so who knows.
 - Very small input files will end up larger after compression. Take the above format for a two byte input like 'Hi', after compression this has a 7 byte header of the table length 2 (as u8), and then the next six (2 symbols * (3 * u8 per symbol)) bytes of the table, followed by the encoded message that would literally be two bits and finally one byte to record that the last six bits of the encoded message is to be ignored on decode.

### How does it perform on real world test data?

**TODO**
