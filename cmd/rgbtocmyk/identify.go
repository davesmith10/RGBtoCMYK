package main

import (
	"fmt"
	"os"

	"github.com/davesmith10/RGBtoCMYK/internal/color"
	"github.com/davesmith10/RGBtoCMYK/internal/jpeg"
	"github.com/spf13/cobra"
)

var identifyCmd = &cobra.Command{
	Use:   "identify [file]",
	Short: "Inspect image and ICC profile info",
	Args:  cobra.ExactArgs(1),
	RunE:  runIdentify,
}

func init() {
	rootCmd.AddCommand(identifyCmd)
}

func runIdentify(cmd *cobra.Command, args []string) error {
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	info, err := jpeg.GetInfo(data)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	fmt.Printf("File:       %s\n", path)
	fmt.Printf("Dimensions: %d x %d\n", info.Width, info.Height)
	fmt.Printf("Components: %d\n", info.NumComponents)
	fmt.Printf("Color space: %s\n", info.ColorSpace)
	fmt.Printf("File size:  %d bytes (%.1f MB)\n", len(data), float64(len(data))/(1024*1024))

	if info.ICC != nil {
		pi, err := color.ParseProfileInfo(info.ICC)
		if err != nil {
			fmt.Printf("ICC profile: present (%d bytes) but invalid: %v\n", len(info.ICC), err)
		} else {
			fmt.Printf("ICC profile: %d bytes\n", len(info.ICC))
			fmt.Printf("  Version:     %s\n", pi.Version)
			fmt.Printf("  Color space: %s\n", color.ColorSpaceName(pi.ColorSpace))
			fmt.Printf("  PCS:         %s\n", color.ColorSpaceName(pi.PCS))
			fmt.Printf("  Class:       %s\n", color.ProfileClassName(pi.Class))
		}
	} else {
		fmt.Println("ICC profile: none")
	}

	return nil
}
