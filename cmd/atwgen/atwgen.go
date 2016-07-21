package main

import (
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
)

func getImage(path string) image.Image {
	imageFd, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer imageFd.Close()

	image, _, err := image.Decode(imageFd)
	if err != nil {
		panic(err)
	}
	return image
}

func main() {
	templateImage := getImage("template.png")
	templateMask := getImage("template_mask.png")
	backgroundImage := getImage("background")

	destinationImage := image.NewNRGBA(templateImage.Bounds())

	// put base template into our destination
	draw.Draw(
		destinationImage,
		destinationImage.Bounds(),
		templateImage,
		image.ZP,
		draw.Src,
	)

	draw.DrawMask(
		destinationImage,
		destinationImage.Bounds(),
		backgroundImage,
		image.ZP,
		templateMask,
		image.ZP,
		draw.Over,
	)

	fd, err := os.Create("out.png")
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	err = png.Encode(fd, destinationImage)
	if err != nil {
		panic(err)
	}
}
