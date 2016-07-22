package main

import (
	"encoding/json"
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

type backgroundConf struct {
	Path      string
	Placement placement
}

func writeBackground(backgroundConfig backgroundConf, destinationImage draw.Image) draw.Image {
	templateMask := mustGetImage("template_mask.png")
	if backgroundConfig.Path == "" {
		backgroundConfig.Path = "background"
	}
	backgroundImage := mustGetImage(backgroundConfig.Path)

	// resize to the size of the template
	backgroundImage = resize.Resize(
		// scale to the width of the template
		comicWidth,
		0,
		backgroundImage,
		resize.Bilinear,
	)
	backgroundImageHeight := backgroundImage.Bounds().Dy()
	backgroundSegmentSize := backgroundImageHeight / 5
	backgroundStartingY := (int(backgroundConfig.Placement) - 1) * backgroundSegmentSize

	// if the placement makes the image not fully fit in the template, align the bottom edge with the bottom edge of the template
	if destinationImageHeight, pixelsInImage := destinationImage.Bounds().Dy(), backgroundImageHeight-backgroundStartingY; pixelsInImage < destinationImageHeight {
		backgroundStartingY = backgroundImageHeight - destinationImageHeight
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

func writeTextList(textConfigList []textConf, destinationImage draw.Image) draw.Image {
	// copy for easier semantics
	destinationImage = copyImage(destinationImage)

	for i, textConfig := range textConfigList {
		// writing an empty string still does a background, so let's not do that
		if textConfig.Text == "" {
			continue
		}
		// create text image for panel
		textImage := writeSingleText(textConfig)
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
	Text      string    `json:"text"`
	Placement placement `json:"placement"`
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
	drawDistance := (&font.Drawer{Face: fontFace}).MeasureString(textConfig.Text)

	// get the baseline start point based on the placement
	if textConfig.Placement == noPlacement {
		textConfig.Placement = choosePlacement(textConfig.Text)
	}
	baselineStartPoint := baselinePointForPlacement(textConfig.Placement)

	// add some variance to the starting baseline
	startPoint := image.Pt(
		baselineStartPoint.X+offsetX(textConfig.Text),
		baselineStartPoint.Y+offsetY(textConfig.Text),
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
	drawer.DrawString(textConfig.Text)

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

type panelConf struct {
	Text      string `json:"text"`
	Placement string `json:"placement"`
}

type comicBackgroundConf struct {
	Path      string `json:"path"`
	Placement string `json:"placement"`
}

type config struct {
	PanelConfigList  []panelConf         `json:"panels"`
	BackgroundConfig comicBackgroundConf `json:"background"`
}

func panelConfList2textConfList(panelConfigList []panelConf) []textConf {
	textConfigList := make([]textConf, 0, len(panelConfigList))
	for _, panelConfig := range panelConfigList {
		place := noPlacement

		switch panelConfig.Placement {
		case "top":
			place = topPlacement
		case "top-middle":
			place = topMiddlePlacement
		case "middle":
			place = middlePlacement
		case "bottom-middle":
			place = bottomMiddlePlacement
		case "bottom":
			place = bottomPlacement
		}

		textConfigList = append(
			textConfigList,
			textConf{
				Text:      panelConfig.Text,
				Placement: place,
			},
		)
	}
	return textConfigList
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			os.Exit(1)
		}
	}()

	conf := config{}
	configFd, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	err = json.NewDecoder(configFd).Decode(&conf)
	if err != nil {
		panic(err)
	}
	fmt.Println(conf)

	destinationImage := writeTextList(
		panelConfList2textConfList(conf.PanelConfigList),
		writeBackground(backgroundConf{Placement: topMiddlePlacement, Path: "background"}, generateBasicTemplate()),
	)

	err = writeImage("out.png", destinationImage)
	if err != nil {
		panic(err)
	}
}
