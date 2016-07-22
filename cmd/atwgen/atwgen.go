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
	"github.com/nfnt/resize"
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

	// resize to the size of the template
	backgroundImage = resize.Resize(
		// scale to the width of the template
		comicWidth,
		0,
		backgroundImage,
		resize.Bilinear,
	)
	// this centers the background image such that the center of it (vertically) is in the center of the comic
	backgroundStartingY := (backgroundImage.Bounds().Dy() - comicHeight) / 2
	if backgroundStartingY < 0 {
		// this probably looks bad because it means the image is shorted than the comic
		backgroundStartingY = 0
	}

	draw.DrawMask(
		destinationImage,
		destinationImage.Bounds(),
		backgroundImage,
		image.Pt(0, backgroundStartingY),
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
	comicWidth            = 720
	comicHeight           = 275
	fontSize              = 14.0
	textBackgroundPadding = 3
	numPlacements         = 5
	baselineX             = 30
)

func baselinePointForPlacement(place placement) image.Point {
	segmentSize := panelRectangle.Dy() / numPlacements

	// multiply the number of segments above (which corresponds to the number of the placement minus 1)
	// then add half a segment to put it in the middle of that (this helps put it not right next to edges)
	baselineY := (int(place)-1)*segmentSize + segmentSize/2
	return image.Pt(baselineX, baselineY)
}

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
		textImage := writeSingleText(textConf{text: text})
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

// between -10 and 10 pixel offset
const offsetBound = 21

func hashString(text string, reduce func(left, right rune) rune) int {
	var accumulator rune
	for _, ch := range text {
		accumulator = reduce(accumulator, ch)
	}
	return int(accumulator)
}

func choosePlacement(text string) placement {
	hash := hashString(text, func(left, right rune) rune { return left | right })
	// mod by the number of placements and then add one to not get noPlacement
	return placement((hash % numPlacements) + 1)
}

func offset(text string, reduce func(left, right rune) rune) int {
	hash := hashString(text, reduce)
	return int(hash%offsetBound - (offsetBound / 2))
}

func offsetX(text string) int {
	return offset(text, func(left, right rune) rune { return left * right })
}

func offsetY(text string) int {
	return offset(text, func(left, right rune) rune { return left + right })
}

type placement int

const (
	noPlacement placement = iota
	topPlacement
	topMiddlePlacement
	middlePlacement
	bottomMiddlePlacement
	bottomPlacement
)

type textConf struct {
	text  string
	place placement
}

func writeSingleText(textConfig textConf) draw.Image {
	// create a panel image to draw our text to
	destinationImage := image.NewNRGBA(panelRectangle)

	// create font face for our font
	fontFace := truetype.NewFace(
		getFont(),
		&truetype.Options{Size: fontSize},
	)

	// create a drawer to draw the text starting at the baseline point, in the font and measure the distance of the string
	drawDistance := (&font.Drawer{Face: fontFace}).MeasureString(textConfig.text)

	// get the baseline start point based on the placement
	if textConfig.place == noPlacement {
		textConfig.place = choosePlacement(textConfig.text)
	}
	baselineStartPoint := baselinePointForPlacement(textConfig.place)

	// add some variance to the starting baseline
	startPoint := image.Pt(
		baselineStartPoint.X+offsetX(textConfig.text),
		baselineStartPoint.Y+offsetY(textConfig.text),
	)

	borderRect := withPadding(
		// create a rectangle for the border
		image.Rect(
			// top left x is the same as the baseline
			startPoint.X,
			// top left y is the baseline y moved up by the ascent of the font (the distance between the baseline and the top of the font)
			startPoint.Y-fontFace.Metrics().Ascent.Round(),
			// bottom right x is the baseline start point x plus the calculated distance for drawing
			startPoint.X+drawDistance.Round(),
			// bottom right y is the same as the baseline
			startPoint.Y,
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
	drawer := &font.Drawer{
		Dst:  destinationImage,
		Src:  image.Black,
		Face: fontFace,
		Dot: fixed.P(
			startPoint.X,
			startPoint.Y,
		),
	}
	drawer.DrawString(textConfig.text)

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
	destinationImage := writeTextList([]string{"foo", "bar", "baz"}, writeBackground(generateBasicTemplate()))
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
