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

func generateBasicTemplate() draw.Image {
	templateImage := getImage("template.png")
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
	templateMask := getImage("template_mask.png")
	backgroundImage := getImage("background")

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

func writeImage(path string, image image.Image) error {
	fd, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	return png.Encode(fd, image)
}

func main() {
	destinationImage := writeBackground(generateBasicTemplate())

	err = writeImage("out.png", destinationImage)
	if err != nil {
		panic(err)
	}
}
