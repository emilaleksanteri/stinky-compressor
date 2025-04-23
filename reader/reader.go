package reader

import (
	"errors"
	"io"
)

const END_OF_READING = byte('f')

type BitReader struct {
	reader         io.Reader
	buffer         byte // 8 bits at a time
	bitsRemaining  int
	paddingSize    int
	totalBytesRead int64
	encodedSize    int64
	finished       bool
}

func (b *BitReader) Next() bool {
	if b.finished {
		return false
	}

	return true
}

func NewBitReader(reader io.Reader, encodedSize int64, paddingSize int) *BitReader {
	return &BitReader{
		reader:        reader,
		bitsRemaining: 0,
		paddingSize:   paddingSize,
		encodedSize:   encodedSize,
	}
}

func (b *BitReader) ReadBit() (byte, error) {
	if b.bitsRemaining == 0 {
		if b.totalBytesRead >= b.encodedSize {
			b.finished = true
			return END_OF_READING, nil
		}

		buf := make([]byte, 1)
		_, err := b.reader.Read(buf)
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				b.finished = true
				return END_OF_READING, nil
			default:
				return END_OF_READING, err
			}
		}

		b.buffer = buf[0]
		b.bitsRemaining = 8
		b.totalBytesRead++

		if b.totalBytesRead == b.encodedSize {
			// we substract padding from our buffer expected read size for the case that we have something like
			// 11110000 with our buffer being 4 bits in this example
			// now, we can still use our grab first bit method by total size offset below
			// but we stop reading before we get to the 0000 since our bitsRemaining is 4 instead of 8

			b.bitsRemaining -= b.paddingSize
			if b.bitsRemaining <= 0 {
				b.finished = true
				return END_OF_READING, nil
			}
		}
	}

	// our buffer is always 8 bits, what we do in the two lines here is we grab the first bit of the buffer
	// once we have that bit we left shift the buffer to remove the first bit and move everything forward by 1
	// as we append a 0 bit to the end
	// e.g., if we have 11111111 as our buffer bits, we do this:
	// 1 . take 1 from the front
	// 2. left shift by 1 and our buffer becomes 11111110
	bit := (b.buffer >> 7) & 1
	b.buffer <<= 1

	b.bitsRemaining--

	return bit, nil
}
