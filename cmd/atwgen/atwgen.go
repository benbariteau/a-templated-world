package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"os"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func mustGetImage(path string) image.Image {
	image, err := getImage(path)
	if err != nil {
		panic(err)
	}
	return image
}

func getImage(path string) (image.Image, error) {
	imageFd, err := os.Open(path)
	if err != nil {
		return image.Black, err
	}
	defer imageFd.Close()

	img, _, err := image.Decode(imageFd)
	if err != nil {
		return image.Black, err
	}
	return img, nil
}

func generateBasicTemplate() draw.Image {
	templateImage := mustGetImage("template.png")
	destinationImage := image.NewNRGBA(templateImage.Bounds())

	// put base template into our destination
	draw.Draw(
		destinationImage,
		destinationImage.Bounds(),
		templateImage,
		image.ZP,
		draw.Src,
	)
	return destinationImage
}

func writeBackground(destinationImage draw.Image) draw.Image {
	templateMask := mustGetImage("template_mask.png")
	backgroundImage := mustGetImage("background")

	draw.DrawMask(
		destinationImage,
		destinationImage.Bounds(),
		backgroundImage,
		image.ZP,
		templateMask,
		image.ZP,
		draw.Over,
	)

	return destinationImage
}

func getFont() *truetype.Font {
	fontFd, err := os.Open("Loveletter_TW.ttf")
	if err != nil {
		panic(err)
	}

	fontBytes, err := ioutil.ReadAll(fontFd)
	if err != nil {
		panic(err)
	}

	font, err := truetype.Parse(fontBytes)
	if err != nil {
		panic(err)
	}

	return font
}

const (
	fontSize              = 14.0
	baselineX             = 30
	baselineY             = 120
	textBackgroundPadding = 3
)

func withPadding(rect image.Rectangle, padding int) image.Rectangle {
	return image.Rect(
		rect.Min.X-padding,
		rect.Min.Y-padding,
		rect.Max.X+padding,
		rect.Max.Y+padding,
	)
}

var panelToTopLeft = map[int]image.Point{
	0: image.Pt(13, 37),
	1: image.Pt(254, 37),
	2: image.Pt(493, 38),
}

var panelRectangle = image.Rect(
	0, 0,
	212, 216,
)

var panelToRectangle = func() map[int]image.Rectangle {
	m := make(map[int]image.Rectangle)
	for panelNumber, topLeft := range panelToTopLeft {
		m[panelNumber] = panelRectangle.Add(topLeft)
	}
	return m
}()

func copyImage(img image.Image) draw.Image {
	// create a new image
	copyTo := image.NewNRGBA(img.Bounds())

	// copy stuff to that image
	draw.Draw(
		copyTo,
		copyTo.Bounds(),
		img,
		image.ZP,
		draw.Src,
	)
	return copyTo
}

func writeTextList(textConfig []string, destinationImage draw.Image) draw.Image {
	// copy for easier semantics
	destinationImage = copyImage(destinationImage)

	for i, text := range textConfig {
		// writing an empty string still does a background, so let's not do that
		if text == "" {
			continue
		}
		// create text image for panel
		textImage := writeSingleText(text)
		// write text image on top of panel
		draw.DrawMask(
			destinationImage,
			panelToRectangle[i],
			textImage,
			image.ZP,
			image.Black,
			image.ZP,
			draw.Over,
		)
	}
	return destinationImage
}

func writeSingleText(text string) draw.Image {
	// create a panel image to draw our text to
	destinationImage := image.NewNRGBA(panelRectangle)

	// create font face for our font
	fontFace := truetype.NewFace(
		getFont(),
		&truetype.Options{Size: fontSize},
	)

	// create the start point for our baseline for the text
	startPoint := fixed.P(baselineX, baselineY)

	// create a drawer to draw the text starting at the baseline point, in the font
	drawer := &font.Drawer{
		Face: fontFace,
		Dot:  startPoint,
	}

	// measure the distance of the string
	drawDistance := drawer.MeasureString(text)
	borderRect := withPadding(
		// create a rectangle for the border
		image.Rect(
			// top left x is the same as the baseline
			baselineX,
			// top left y is the baseline y moved up by the ascent of the font (the distance between the baseline and the top of the font)
			baselineY-fontFace.Metrics().Ascent.Round(),
			// bottom right x is the baseline start point x plus the calculated distance for drawing
			baselineX+drawDistance.Round(),
			// bottom right y is the same as the baseline
			baselineY,
		),
		// pad that rectangle
		textBackgroundPadding,
	)

	// draw the background rectangle into the destination image in white
	draw.DrawMask(
		destinationImage,
		destinationImage.Bounds(),
		image.White,
		image.ZP,
		borderRect,
		image.ZP,
		draw.Over,
	)

	// draw the text, in black to the return value
	drawer.Dst = destinationImage
	drawer.Src = image.Black
	drawer.DrawString(text)

	return destinationImage
}

func writeImage(path string, image image.Image) error {
	fd, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	return png.Encode(fd, image)
}

func main() {
	destinationImage := writeTextList([]string{"foo", "", "baz"}, writeBackground(generateBasicTemplate()))
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			os.Exit(1)
		}
	}()

	err := writeImage("out.png", destinationImage)
	if err != nil {
		panic(err)
	}
}
