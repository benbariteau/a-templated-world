package main

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
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

	img, err := gen(dirPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	writeImgToFile("out.png", img)
}

func gen(dirPath string) (img draw.Image, err error) {
	captions, pictures, err := getSrcData(dirPath)
	if err != nil {
		return
	}
	if len(captions) == 0 {
		err = errors.New("No caption file found or empty caption file")
		return
	}
	text := captions[0]

	img = image.NewNRGBA(image.Rect(0, 0, 1000, 1000))
	draw.Draw(img, image.Rect(0, 0, 1000, 1000), image.NewUniform(color.Black), image.Point{0, 0}, draw.Over)
	draw.Draw(img, image.Rect(0, 0, 100, 100), pictures[0], image.Point{0, 0}, draw.Over)

	fontContext, err := createFontContext(img)
	if err != nil {
		return
	}

	fontContext.DrawString(string(text), freetype.Pt(100, 100))

	return
}

func writeImgToFile(filename string, img image.Image) error {
	outimg, err := os.Create(filename)
	if err != nil {
		return errors.New(fmt.Sprint("Unable to open image for writing", err))
	}
	defer outimg.Close()

	err = png.Encode(outimg, img)
	if err != nil {
		return errors.New("Unable to write image")
	}
	return nil
}

func createFontContext(dst draw.Image) (fontContext *freetype.Context, err error) {
	fontContext = freetype.NewContext()

	fontBytes, err := ioutil.ReadFile("/Users/bariteau/Downloads/love_letter_tw/Lovelt__.ttf")
	if err != nil {
		err = errors.New("Unable to read font file")
		return
	}
	font, err := freetype.ParseFont(fontBytes)

	if err != nil {
		err = errors.New("Unable to parse font file")
		return
	}

	fontContext.SetFont(font)
	fontContext.SetFontSize(12.0)
	fontContext.SetSrc(image.NewUniform(color.White))
	fontContext.SetDst(dst)
	fontContext.SetClip(dst.Bounds())
	return
}

func getSrcData(dirPath string) (captions []string, pictures []image.Image, err error) {
	captionFile, pictureFiles, err := getSrcFiles(dirPath)
	if err != nil {
		return
	}
	captionFileScanner := bufio.NewScanner(captionFile)
	for captionFileScanner.Scan() {
		captions = append(captions, captionFileScanner.Text())
	}
	err = captionFileScanner.Err()

	pictures = make([]image.Image, len(pictureFiles))
	for i, pictureFile := range pictureFiles {
		pictures[i], _, err = image.Decode(pictureFile)
		if err != nil {
			return
		}
	}
	return
}

var imgSrcFilenamesPattern = regexp.MustCompile("([1-3])[.](png|PNG)")

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

	picturesFileMap := make(map[string]*os.File)
	for _, fileinfo := range fileinfos {
		name := fileinfo.Name()
		fmt.Println(name)
		if strings.HasPrefix(name, "caption") {
			if captions == nil {
				captions, err = os.Open(path.Join(dirPath, name))
				if err != nil {
					captions = nil
				}
			}
		} else if matches := imgSrcFilenamesPattern.FindStringSubmatch(name); matches != nil {
			file, err := os.Open(path.Join(dirPath, name))
			if err != nil {
				continue
			}
			picturesFileMap[matches[1]] = file
		}
	}

	pictures = make([]*os.File, len(picturesFileMap))
	for i := range pictures {
		pictures[i] = picturesFileMap[strconv.Itoa(i+1)]
	}
	return
}
