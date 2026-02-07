package main

import (
	"fmt"
	"os"

	"github.com/davesmith10/RGBtoCMYK/internal/color"
	"github.com/davesmith10/RGBtoCMYK/internal/pipeline"
	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert an RGB JPEG to CMYK JPEG",
	RunE:  runConvert,
}

func init() {
	convertCmd.Flags().StringP("input", "i", "", "Input RGB JPEG file")
	convertCmd.Flags().StringP("output", "o", "", "Output CMYK JPEG file")
	convertCmd.Flags().String("profile", "", "CMYK ICC profile path")
	convertCmd.Flags().String("src-profile", "", "Source RGB ICC profile override")
	convertCmd.Flags().Int("quality", 85, "JPEG quality (1-100)")
	convertCmd.Flags().Int("cmy-reduction", 15, "Quality reduction for CMY channels vs K")
	convertCmd.Flags().String("intent", "perceptual", "Rendering intent (perceptual, relative, saturation, absolute)")
	convertCmd.MarkFlagRequired("input")
	convertCmd.MarkFlagRequired("output")
	convertCmd.MarkFlagRequired("profile")
	rootCmd.AddCommand(convertCmd)
}

func runConvert(cmd *cobra.Command, args []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	profilePath, _ := cmd.Flags().GetString("profile")
	srcProfilePath, _ := cmd.Flags().GetString("src-profile")
	quality, _ := cmd.Flags().GetInt("quality")
	cmyReduction, _ := cmd.Flags().GetInt("cmy-reduction")
	intentStr, _ := cmd.Flags().GetString("intent")

	intent, err := color.ParseIntent(intentStr)
	if err != nil {
		return err
	}

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	dstProfile, err := color.LoadProfile(profilePath)
	if err != nil {
		return fmt.Errorf("loading CMYK profile: %w", err)
	}

	var srcProfile []byte
	if srcProfilePath != "" {
		srcProfile, err = color.LoadProfile(srcProfilePath)
		if err != nil {
			return fmt.Errorf("loading source profile: %w", err)
		}
	}

	opts := pipeline.Options{
		SrcProfileOverride: srcProfile,
		DstProfile:         dstProfile,
		Quality:            quality,
		CMYReduction:       cmyReduction,
		Intent:             intent,
	}

	result, err := pipeline.Run(inputData, opts)
	if err != nil {
		return fmt.Errorf("conversion: %w", err)
	}

	if err := os.WriteFile(outputPath, result.Data, 0644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("Converted %dx%d RGB → CMYK\n", result.SrcWidth, result.SrcHeight)
	fmt.Printf("Input:  %s (%d bytes)\n", inputPath, len(inputData))
	fmt.Printf("Output: %s (%d bytes)\n", outputPath, len(result.Data))

	return nil
}
