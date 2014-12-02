package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"code.google.com/p/freetype-go/freetype"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Printf("usage:\t%v [path/to/sources]\n", args[0])
		return
	}

	dirPath := os.Args[1]

	err := gen(dirPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func gen(dirPath string) error {
	captionFile, _, err := getSrcFiles(dirPath)
	if err != nil {
		return err
	}
	if captionFile == nil {
		return errors.New("No caption file found")
	}
	text, err := ioutil.ReadAll(captionFile)
	if err != nil {
		return err
	}

	fontBytes, err := ioutil.ReadFile("/Users/bariteau/Downloads/love_letter_tw/Lovelt__.ttf")
	if err != nil {
		return errors.New("Unable to read font file")
	}
	font, err := freetype.ParseFont(fontBytes)

	if err != nil {
		return errors.New("Unable to parse font file")
	}

	img := image.NewNRGBA(image.Rect(0, 0, 1000, 1000))
	draw.Draw(img, image.Rect(0, 0, 1000, 1000), image.NewUniform(color.Black), image.Point{0, 0}, draw.Over)
	fontContext := freetype.NewContext()
	fontContext.SetFont(font)
	fontContext.SetFontSize(12.0)
	fontContext.SetSrc(image.NewUniform(color.White))
	fontContext.SetDst(img)
	fontContext.SetClip(img.Bounds())
	fontContext.DrawString(string(text), freetype.Pt(100, 100))
	outimg, err := os.Create("out.png")
	defer func() {
		er := outimg.Close()
		if er != nil {
			err = er
		}
	}()
	if err != nil {
		return errors.New(fmt.Sprint("Unable to open image for writing", err))
	}
	err = png.Encode(outimg, img)
	if err != nil {
		return errors.New("Unable to write image")
	}
	return nil
}

func getSrcFiles(dirPath string) (captions *os.File, pictures []*os.File, err error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return
	}

	dirInfo, err := dir.Stat()
	if err != nil {
		return
	}

	if !dirInfo.IsDir() {
		errors.New(fmt.Sprint("Not a directory: ", dirPath))
	}

	fileinfos, err := dir.Readdir(0)
	if err != nil {
		return
	}

	for _, fileinfo := range fileinfos {
		name := fileinfo.Name()
		if strings.HasPrefix(name, "caption") {
			if captions == nil {
				captions, err = os.Open(path.Join(dirPath, name))
				if err != nil {
					captions = nil
				}
			}
			continue
		}
	}
	return
}
