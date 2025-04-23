package writer

import "io"

type BitWriter struct {
	writer      io.Writer
	buffer      byte // we write 8 bits at a time which is a byte
	bitsWritten int
}

func NewBitWriter(w io.Writer) *BitWriter {
	return &BitWriter{
		writer: w,
	}
}

func (b *BitWriter) WriteBits(path uint64, bitSize int) error {
	for pos := bitSize - 1; pos >= 0; pos-- {
		bit := (path >> uint(pos)) & 1 // gets first bit in path i.e. if path is 01001 we would get 0 at first iter since we right shift by size - 1

		// if we have 111 we shift L shift by one making it 1110, then with | op we append our bit to the L shiften position,
		// so if our bit is 1 b.buffer would become 1111
		b.buffer = (b.buffer << 1) | byte(bit)

		b.bitsWritten++ // keep track to see when we have 8 bits or a full byte

		if b.bitsWritten == 8 {
			_, err := b.writer.Write([]byte{b.buffer})
			if err != nil {
				return err
			}

			b.buffer = 0
			b.bitsWritten = 0
		}
	}

	return nil
}

// returned int is a encoding padding size which will be needed when decoding the file
func (b *BitWriter) Flush() (int, error) {
	paddingSize := 0
	if b.bitsWritten > 0 {
		// since we can only write 8 bits at a time, if the remaining bits are not exactly a byte, we need to add some padding
		// to make a full byte
		paddingSize = 8 - b.bitsWritten
		b.buffer <<= byte(paddingSize)

		_, err := b.writer.Write([]byte{b.buffer})
		if err != nil {
			return paddingSize, err
		}

		b.buffer = 0
		b.bitsWritten = 0

	}

	return paddingSize, nil
}
