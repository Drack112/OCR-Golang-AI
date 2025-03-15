package image

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"image"
	"image/draw"
	"io"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"golang.design/x/clipboard"
	drawX "golang.org/x/image/draw"
)

type Dimensions struct {
	Width, Height int
}

type TranslatorImage struct {
	Image      *image.RGBA
	Hash       string
	Dimensions Dimensions
	size       int
}

func Open(file string, url, clip bool) TranslatorImage {
	var img image.Image
	var h hash.Hash
	var size int

	if clip {
		var err error

		err = clipboard.Init()
		if err != nil {
			log.Fatal(err)
		}

		imgByte := clipboard.Read(clipboard.FmtImage)
		if imgByte == nil {
			log.Fatal("No image found in clipboard")
		}

		var buf bytes.Buffer
		tee := io.TeeReader(bytes.NewReader(imgByte), &buf)
		img, _, err = image.Decode(tee)
		size = buf.Len()
		if err != nil {
			log.Fatalf("Image decode error: %v", err)
		}

		h = sha256.New()
		if _, err := io.Copy(h, &buf); err != nil {
			log.Fatalf("Hash error: %v", err)
		}
	} else if url {
		resp, err := http.Get(file)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		var buf bytes.Buffer
		tee := io.TeeReader(resp.Body, &buf)

		img, _, err = image.Decode(tee)
		size = buf.Len()
		if err != nil {
			log.Fatalf("Image decode error: %v", err)
		}

		h = sha256.New()
		if _, err := io.Copy(h, &buf); err != nil {
			log.Fatalf("Hash error: %v", err)
		}
	} else {
		f, err := os.Open(filepath.ToSlash(file))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		var buf bytes.Buffer
		tee := io.TeeReader(f, &buf)

		img, _, err = image.Decode(tee)
		size = buf.Len()
		if err != nil {
			log.Fatalf("Image decode error: %v", err)
		}

		h = sha256.New()
		if _, err := io.Copy(h, &buf); err != nil {
			log.Fatalf("Hash error: %v", err)
		}
	}

	hashInBytes := h.Sum(nil)
	hashStr := hex.EncodeToString(hashInBytes)
	dims := getDimensions(img)
	imageRGBA := convertToRGBA(img)
	newImg := TranslatorImage{
		Image:      imageRGBA,
		size:       size,
		Hash:       hashStr,
		Dimensions: dims,
	}
	log.Debugf("Hash: %v", hashStr)
	log.Debugf("Image Dimensions: %v", dims)
	newImg.resize()
	return newImg
}

func convertToRGBA(img image.Image) *image.RGBA {
	b := img.Bounds()
	imgB := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(imgB, imgB.Bounds(), img, b.Min, draw.Src)
	return imgB
}

func (img *TranslatorImage) resize() {
	desiredSize := 20000000

	if img.size <= desiredSize {
		return
	}

	log.Info("Resizing Image")
	ratio := img.size / desiredSize

	dst := image.NewRGBA(image.Rect(0, 0, img.Dimensions.Width/ratio, img.Dimensions.Height/ratio))

	drawX.CatmullRom.Scale(dst, dst.Rect, img.Image, img.Image.Bounds(), draw.Over, nil)
	img.Image = dst
	img.Dimensions = getDimensions(dst)
	log.Debugf("New image dimensions: %v", img.Dimensions)
}

func getDimensions(img image.Image) Dimensions {
	bounds := img.Bounds()
	return Dimensions{
		Width:  bounds.Max.X,
		Height: bounds.Max.Y,
	}
}

func GetRatio(dims Dimensions, maxDim float32) float32 {
	var ratio float32
	if dims.Width > dims.Height {
		ratio = maxDim / float32(dims.Width)
	} else {
		ratio = maxDim / float32(dims.Height)
	}
	return ratio
}
