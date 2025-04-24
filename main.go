package main

import (
	"flag"
	"fmt"
	"os"
	sCError "stinky-compression/error"
	"stinky-compression/file"
	stinkycompressor "stinky-compression/stinky-compressor"
	"time"
)

type config struct {
	debug          bool
	removeSrcFile  bool
	decodeDestFile string
	srcFile        string
}

func main() {
	var cfg config
	flag.BoolVar(&cfg.debug, "debug", false, "Run Stinky-Compressor in debug mode")
	flag.BoolVar(&cfg.removeSrcFile, "remove-src", false, "Remove source file after compression")
	flag.StringVar(&cfg.decodeDestFile, "decode-dest", "", "Where to save decoded content")
	flag.StringVar(&cfg.srcFile, "src", "", "Source file to compress")
	flag.Parse()

	switch {
	case flag.NArg() == 0:
		if cfg.srcFile == "" {
			err := &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  "Missing 'src' parameter",
			}

			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}

		fileContent, err := file.ReadInputFile(cfg.srcFile)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}

		compTime := time.Now()
		compressedFileName, err := stinkycompressor.WriteCompressionToFile(fileContent, cfg.srcFile, cfg.removeSrcFile, cfg.debug)
		if err != nil {
			panic(err)
		}
		compSince := time.Since(compTime)
		fmt.Printf("(info) compression took: %s\n", compSince)
		fmt.Printf("(info) Compressed file %s saved\n", compressedFileName)
		os.Exit(0)
	case flag.Arg(0) == "decode":
		if cfg.decodeDestFile == "" {
			err := &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  "Missing 'decode-dest' parameter",
			}

			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}

		if cfg.srcFile == "" {
			err := &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  "Missing 'src' parameter",
			}

			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}

		compressedContent, err := file.ReadInputFile(cfg.srcFile)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}

		decTime := time.Now()
		decoded, err := stinkycompressor.DecodeCompressedFile(compressedContent, cfg.debug)
		if err != nil {
			panic(err)
		}
		decSince := time.Since(decTime)
		fmt.Printf("(info) decode took: %s\n", decSince)

		if !file.FileExists(cfg.decodeDestFile) {
			err := file.CreateFile(cfg.decodeDestFile)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				os.Exit(1)
			}
		}

		fileToWrite, err := file.OpenFileWithWritePermissions(cfg.decodeDestFile)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}
		defer fileToWrite.Close()

		_, err = fileToWrite.Write(decoded)
		if err != nil {
			err = &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  fmt.Sprintf("Failed to write decoded content to file: %+v", err),
			}

			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		fmt.Printf("(info) Decoded content saved at %s\n", cfg.decodeDestFile)
		os.Exit(0)

	case flag.Arg(0) == "compare":
		if cfg.decodeDestFile == "" {
			err := &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  "Missing 'decode-dest' parameter",
			}

			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}

		if cfg.srcFile == "" {
			err := &sCError.CompressorError{
				Severity: sCError.COMPRESSOR_ERROR_SEVERITY_ERROR,
				Message:  "Missing 'src' parameter",
			}

			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}

		inputContent, err := file.ReadInputFile(cfg.srcFile)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		compContent, err := file.ReadInputFile(cfg.decodeDestFile)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		if string(inputContent) != string(compContent) {
			fmt.Println("(error) decoded did not match encoded")
			fmt.Printf("(error) decoded len: %d, input len: %d\n", len(compContent), len(inputContent))
			os.Exit(1)
		}

		fmt.Println("(info) Files matched :)")
		os.Exit(0)
	}
}
