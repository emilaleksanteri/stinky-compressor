package stinkycompressor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	sCError "stinky-compression/error"
	sCFile "stinky-compression/file"
	"stinky-compression/huffman"
	"stinky-compression/reader"
	"stinky-compression/writer"
	"strings"
)

const (
	META_SEPARATOR            = '#'
	COMPRESSED_FILE_EXTENSION = "stinkc"
)

type CompressedFileMetaData struct {
	EncodedLen  int                   `json:"e"`
	Dict        huffman.EncodingTable `json:"d"`
	PaddingSize int                   `json:"ps"`
}

func (cfm *CompressedFileMetaData) DecodeEncodingTable() {
	dict := cfm.Dict
	dict.DecodeSafePathMetaFromTable()
	cfm.Dict = dict
}

func deleteFile(filename string) error {
	return os.Remove(filename)
}

func makeCompressedFileName(fromFileName string) string {
	ext := filepath.Ext(fromFileName)
	if ext == "" {
		return fmt.Sprintf("%s.%s", fromFileName, COMPRESSED_FILE_EXTENSION)
	}

	filenameSplit := strings.Split(fromFileName, ext)
	if len(filenameSplit) == 0 {
		return fmt.Sprintf("%s.%s", fromFileName, COMPRESSED_FILE_EXTENSION)
	}

	rawFileName := filenameSplit[0]

	return fmt.Sprintf("%s.%s", rawFileName, COMPRESSED_FILE_EXTENSION)
}

func WriteCompressionToFile(input []byte, filename string, removeOldFile, debug bool) (string, error) {
	encoded, dict := huffman.HuffmanEncoding(input, debug)

	compressedFileName := makeCompressedFileName(filename)

	if !sCFile.FileExists(compressedFileName) {
		if err := sCFile.CreateFile(compressedFileName); err != nil {
			return compressedFileName, err
		}

	}

	file, err := sCFile.OpenFileWithWritePermissions(compressedFileName)
	if err != nil {
		return compressedFileName, err
	}

	defer file.Close()

	buf := []byte{}
	binBuf := bytes.NewBuffer(buf)
	binWriter := writer.NewBitWriter(binBuf)
	for _, enc := range encoded {
		err := binWriter.WriteBits(enc.Path, enc.Size)
		if err != nil {
			return compressedFileName, &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  fmt.Sprintf("failed to write bits: %+v", err),
			}
		}

	}

	padding, err := binWriter.Flush()
	if err != nil {
		return compressedFileName, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to flush bits: %+v", err),
		}
	}

	dict.MakeSafePathMetaForMetadata()

	metadata := CompressedFileMetaData{
		EncodedLen:  binBuf.Len(),
		Dict:        dict,
		PaddingSize: padding,
	}

	metaBts, err := json.Marshal(metadata)
	if err != nil {
		return compressedFileName, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to marshal metadata: %+v", err),
		}
	}

	metaBts = append(metaBts, byte(META_SEPARATOR))
	_, err = file.Write(metaBts)
	if err != nil {
		return compressedFileName, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to write meta bytes to file: %+v", err),
		}
	}

	_, err = binBuf.WriteTo(file)
	if err != nil {
		return compressedFileName, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to write bits to file: %+v", err),
		}
	}

	if removeOldFile {
		err := deleteFile(filename)
		if err != nil {
			return compressedFileName, &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_INFO,
				Message:  fmt.Sprintf("Compression was succesfull but old file could not be removed: %s", err.Error()),
			}
		}
	}

	return compressedFileName, nil
}

func DecodeCompressedFile(content []byte, debug bool) ([]byte, error) {
	metaBtsRead := []byte{}
	readMeta := true
	metaEndsIdx := 0

	for idx, bt := range content {
		if bt == META_SEPARATOR {
			readMeta = false
			metaEndsIdx = idx + 1
			break
		}

		if readMeta {
			metaBtsRead = append(metaBtsRead, bt)
		}
	}

	metaR := CompressedFileMetaData{}
	err := json.Unmarshal(metaBtsRead, &metaR)
	if err != nil {
		return nil, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to unmarshal meta bytes: %+v", err),
		}
	}

	metaR.DecodeEncodingTable()

	binData := make([]byte, metaR.EncodedLen)
	contentLen := len(content)
	binDataIdx := 0
	for {
		if metaEndsIdx == contentLen {
			break
		}

		binData[binDataIdx] = content[metaEndsIdx]
		binDataIdx++
		metaEndsIdx++
	}

	binBuf := bytes.NewBuffer(binData)
	binReader := reader.NewBitReader(binBuf, int64(metaR.EncodedLen), metaR.PaddingSize)

	tree := huffman.TreeFromEncodingTable(metaR.Dict)
	if debug {
		tree.DebugTree()
	}

	var decoded []byte

	head := tree

	for binReader.Next() {
		bit, err := binReader.ReadBit()
		if err != nil {
			return nil, &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  fmt.Sprintf("read bit: %+v", err),
			}
		}

		if bit == reader.END_OF_READING {
			continue
		}

		if bit == 1 {
			head = head.Right
		} else {
			head = head.Left
		}

		if head.Left == nil && head.Right == nil {
			decoded = append(decoded, head.Char)
			head = tree
		}
	}

	return decoded, nil
}
