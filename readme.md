# The Stinky Compressor
Simple compression algorithm implementation using Huffman Coding

Compress:

`go run main.go -src ./input.txt`

Decode:

`go run main.go --src ./input.stinkc --decode-dest input-2.txt decode`

TODO:
- Compress proto binary with one bwt + mft before write (could make it smaller, if not try rle and LZ77 + LZ78 on that as well if possible)
- Try "LZ77 and LZ78" before huffman
- Try RLE before huffman
