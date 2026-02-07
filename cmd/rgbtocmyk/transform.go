package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/davesmith10/RGBtoCMYK/internal/color"
	"github.com/davesmith10/RGBtoCMYK/internal/jpeg"
	"github.com/spf13/cobra"
)

var transformCmd = &cobra.Command{
	Use:   "transform",
	Short: "Color-transform RGB to CMYK (raw output + JSON sidecar)",
	RunE:  runTransform,
}

func init() {
	transformCmd.Flags().StringP("input", "i", "", "Input RGB JPEG file")
	transformCmd.Flags().StringP("output", "o", "", "Output raw CMYK file")
	transformCmd.Flags().String("profile", "", "CMYK ICC profile path")
	transformCmd.Flags().String("src-profile", "", "Source RGB ICC profile override")
	transformCmd.Flags().String("intent", "perceptual", "Rendering intent")
	transformCmd.MarkFlagRequired("input")
	transformCmd.MarkFlagRequired("output")
	transformCmd.MarkFlagRequired("profile")
	rootCmd.AddCommand(transformCmd)
}

type transformMeta struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

func runTransform(cmd *cobra.Command, args []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	profilePath, _ := cmd.Flags().GetString("profile")
	srcProfilePath, _ := cmd.Flags().GetString("src-profile")
	intentStr, _ := cmd.Flags().GetString("intent")

	intent, err := color.ParseIntent(intentStr)
	if err != nil {
		return err
	}

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	decoded, err := jpeg.DecodeRGB(inputData)
	if err != nil {
		return fmt.Errorf("decoding: %w", err)
	}

	dstProfile, err := color.LoadProfile(profilePath)
	if err != nil {
		return err
	}

	srcICC := decoded.ICC
	if srcProfilePath != "" {
		srcICC, err = color.LoadProfile(srcProfilePath)
		if err != nil {
			return err
		}
	}
	if srcICC == nil {
		srcICC = color.EmbeddedSRGB
	}

	xform, err := color.NewTransform(srcICC, dstProfile, intent)
	if err != nil {
		return err
	}
	defer xform.Close()

	cmyk, err := xform.TransformPixels(decoded.Pixels, decoded.Width, decoded.Height)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, cmyk, 0644); err != nil {
		return fmt.Errorf("writing raw CMYK: %w", err)
	}

	// Write JSON sidecar
	meta := transformMeta{
		Width:  decoded.Width,
		Height: decoded.Height,
		Format: "CMYK8",
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := strings.TrimSuffix(outputPath, ".raw") + ".json"
	if err := os.WriteFile(metaPath, metaJSON, 0644); err != nil {
		return fmt.Errorf("writing sidecar: %w", err)
	}

	fmt.Printf("Transformed %dx%d → raw CMYK (%d bytes)\n", decoded.Width, decoded.Height, len(cmyk))
	fmt.Printf("Sidecar: %s\n", metaPath)
	return nil
}
