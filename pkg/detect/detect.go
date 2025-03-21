package detect

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	log "github.com/sirupsen/logrus"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

var borderColors = []color.NRGBA{
	{R: 255, A: 255},         // Red
	{G: 255, A: 255},         // Green
	{B: 255, A: 255},         // Blue
	{R: 255, G: 255, A: 255}, // Yellow
	{R: 255, B: 255, A: 255}, // Violet
	{G: 255, B: 255, A: 255}, // Cyan
}

type TextBlock struct {
	Text       string
	Translated string
	Vertices   []*pb.Vertex
	Color      color.NRGBA
}

var errInvalidVisionPath = errors.New(`path given for Vision API service account key is invalid. Please run the "manga-translator-setup" application to fix it`)

func GetAnnotation(img *image.RGBA) (*pb.TextAnnotation, error) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		log.Errorf("NewImageAnnotatorClient: %v", err)
		if strings.HasPrefix(err.Error(),
			"google: error getting credentials using GOOGLE_APPLICATION_CREDENTIALS environment variable") {
			return nil, errInvalidVisionPath
		}
		return nil, err
	}

	reader := ReaderFromImage(img)
	visionImg, err := vision.NewImageFromReader(reader)
	if err != nil {
		log.Errorf("NewImageFromReader: %v", err)
		return nil, err
	}

	annotation, err := client.DetectDocumentText(ctx, visionImg, &pb.ImageContext{LanguageHints: []string{"ja"}})
	if err != nil {
		log.Errorf("DetectDocumentText: %v", err)
		return nil, err
	}

	if annotation == nil {
		log.Info("No text found")
		return nil, errors.New("no text found")
	} else {
		log.WithField("text", annotation.Text).Info("Detected Text")
		return annotation, nil
	}
}

func OrganizeAnnotation(annotation *pb.TextAnnotation) []TextBlock {
	var blockList []TextBlock
	for _, page := range annotation.Pages {
		for i, block := range page.Blocks {
			var b string
			for _, paragraph := range block.Paragraphs {
				var p string
				for _, word := range paragraph.Words {
					symbols := make([]string, len(word.Symbols))
					for i, s := range word.Symbols {
						symbols[i] = s.Text
					}
					wordText := strings.Join(symbols, "")
					p += wordText
				}
				b += p
			}
			blockList = append(blockList, TextBlock{
				Text:     b,
				Vertices: block.BoundingBox.Vertices,
				Color:    borderColors[i%len(borderColors)],
			})
		}
	}
	return blockList
}

func ReaderFromImage(img *image.RGBA) *bytes.Reader {
	buff := new(bytes.Buffer)

	err := png.Encode(buff, img)
	if err != nil {
		log.Fatalf("Failed to create buffer: %v", err)
	}

	reader := bytes.NewReader(buff.Bytes())
	return reader
}
