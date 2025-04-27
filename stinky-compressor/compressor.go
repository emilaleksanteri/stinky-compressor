package stinkycompressor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"stinky-compression/bwt"
	sCError "stinky-compression/error"
	sCFile "stinky-compression/file"
	"stinky-compression/huffman"
	"stinky-compression/mft"
	proto_data "stinky-compression/proto/proto-data"
	"stinky-compression/reader"
	"stinky-compression/rle"
	"stinky-compression/writer"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
)

const (
	COMPRESSED_FILE_EXTENSION = "stinkc"
)

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
	encoded, frequencyTable, bwtIdx, rleDict := huffman.HuffmanEncoding(input, debug)

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

	metadata := proto_data.CompressedFileMetaData{
		EncodedLen:   int64(binBuf.Len()),
		PaddingSize:  int32(padding),
		OriginalSize: int64(len(input)),
		Frequencies:  huffman.FrequencyTableToProto(frequencyTable),
		BwtIdx:       int32(bwtIdx),
		RleDict:      rleDict,
	}

	metaBts, err := proto.Marshal(&metadata)
	if err != nil {
		return compressedFileName, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to marshal proto: %+v", err),
		}
	}

	metaBtsSize := len(metaBts)

	_, err = file.Write([]byte(fmt.Sprintf("%d#", metaBtsSize)))
	if err != nil {
		return compressedFileName, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to write meta bytes size: %+v", err),
		}

	}

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
	metaEndsIdx := 0
	metaStartIdx := 0
	metaSize := 0

	accSizeStr := ""
	for idx, bt := range content {
		if bt != '#' {
			accSizeStr += string(bt)
		} else {
			metaStartIdx = idx + 1
			metaSizeAtoi, err := strconv.Atoi(accSizeStr)
			if err != nil {
				return nil, &sCError.CompressorError{
					Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
					Message:  fmt.Sprintf("failed to parse meta size: %+v", err),
				}
			}

			metaSize = metaSizeAtoi

			break
		}
	}

	metaBtsRead := make([]byte, metaSize)
	currMetaRead := 0
	metaIdx := 0
	for {
		if currMetaRead == metaSize {
			metaEndsIdx = metaStartIdx
			break
		}

		metaBtsRead[metaIdx] = content[metaStartIdx]
		currMetaRead++
		metaIdx++
		metaStartIdx++
	}

	metaR := &proto_data.CompressedFileMetaData{}
	err := proto.Unmarshal(metaBtsRead, metaR)
	if err != nil {
		return nil, &sCError.CompressorError{
			Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
			Message:  fmt.Sprintf("failed to unmarshal meta bytes: %+v", err),
		}
	}

	binData := make([]byte, metaR.EncodedLen)
	binDataIdx := 0
	bytesRead := int64(0)
	for bytesRead < metaR.EncodedLen {

		binData[binDataIdx] = content[metaEndsIdx]
		binDataIdx++
		metaEndsIdx++
		bytesRead++
	}

	binBuf := bytes.NewBuffer(binData)
	binReader := reader.NewBitReader(binBuf, metaR.GetEncodedLen(), int(metaR.GetPaddingSize()))

	frequencyTable := huffman.ProtoFrequenciesToFrequencyTable(metaR.GetFrequencies())
	tree := huffman.TreeFromFrequencies(frequencyTable)
	head := tree

	decoded := []byte{}

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

	mftDecoded := mft.DecodeMft(decoded)
	rleDecoded := rle.DecodeRle(mftDecoded, metaR.RleDict)
	bwtDecoded := bwt.DecodeBwt(rleDecoded, int(metaR.BwtIdx))

	return bwtDecoded, nil
}
