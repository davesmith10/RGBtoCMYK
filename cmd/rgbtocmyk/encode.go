package main

import (
	"fmt"
	"os"

	"github.com/davesmith10/RGBtoCMYK/internal/jpeg"
	"github.com/spf13/cobra"
)

var encodeCmd = &cobra.Command{
	Use:   "encode",
	Short: "Encode raw CMYK data to JPEG",
	RunE:  runEncode,
}

func init() {
	encodeCmd.Flags().StringP("input", "i", "", "Input raw CMYK file")
	encodeCmd.Flags().StringP("output", "o", "", "Output CMYK JPEG file")
	encodeCmd.Flags().String("icc", "", "ICC profile to embed")
	encodeCmd.Flags().Int("width", 0, "Image width")
	encodeCmd.Flags().Int("height", 0, "Image height")
	encodeCmd.Flags().Int("quality", 85, "JPEG quality (1-100)")
	encodeCmd.Flags().Int("cmy-reduction", 15, "Quality reduction for CMY channels")
	encodeCmd.MarkFlagRequired("input")
	encodeCmd.MarkFlagRequired("output")
	encodeCmd.MarkFlagRequired("width")
	encodeCmd.MarkFlagRequired("height")
	rootCmd.AddCommand(encodeCmd)
}

func runEncode(cmd *cobra.Command, args []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	iccPath, _ := cmd.Flags().GetString("icc")
	width, _ := cmd.Flags().GetInt("width")
	height, _ := cmd.Flags().GetInt("height")
	quality, _ := cmd.Flags().GetInt("quality")
	cmyReduction, _ := cmd.Flags().GetInt("cmy-reduction")

	pixels, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	expected := width * height * 4
	if len(pixels) != expected {
		return fmt.Errorf("expected %d bytes for %dx%d CMYK, got %d", expected, width, height, len(pixels))
	}

	var icc []byte
	if iccPath != "" {
		icc, err = os.ReadFile(iccPath)
		if err != nil {
			return fmt.Errorf("reading ICC profile: %w", err)
		}
	}

	encoded, err := jpeg.EncodeCMYK(pixels, width, height, icc, jpeg.EncoderOptions{
		Quality:      quality,
		CMYReduction: cmyReduction,
	})
	if err != nil {
		return fmt.Errorf("encoding: %w", err)
	}

	if err := os.WriteFile(outputPath, encoded, 0644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("Encoded %dx%d CMYK → %s (%d bytes)\n", width, height, outputPath, len(encoded))
	return nil
}
