package main

import (
	"flag"
	"os"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/unit"

	"github.com/Drack112/Anime-OCR-Translator/pkg/config"
	imageW "github.com/Drack112/Anime-OCR-Translator/pkg/image"
	"github.com/Drack112/Anime-OCR-Translator/pkg/window"
	log "github.com/sirupsen/logrus"
)

var maxDim float32 = 1000 // hard-coded

func main() {
	// Set up logging.
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)

	settings := config.Path()
	logPath := filepath.Join(settings, "mtl-logrus.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err == nil {
		log.SetOutput(f)
	} else {
		log.Warning("Failed to log to file, using default stderr")
	}
	defer f.Close()

	// Parse flags.
	urlImagePtr := flag.Bool("url", false, "Use an image from a URL instead of a local file.")
	clipImagePtr := flag.Bool("clip", false, "Use an image from the clipboard.") // overrides url
	flag.Parse()
	log.Infof("Use URL image: %v", *urlImagePtr)
	log.Infof("Use clipboard image: %v", *clipImagePtr)

	// Set up config, create new config if necessary.
	var cfg config.File
	config.Setup(settings, &cfg)

	// Open/download selected image and get its info.
	if len(flag.Args()) == 0 && !*clipImagePtr {
		log.Fatal("No path or URL given.")
	}
	var imgPath []string
	if !*clipImagePtr {
		imgPath = flag.Args()
		log.Infof("All Selected Image(s): %v", imgPath)
	} else {
		// Need a single element in the array so that it will try to open 1 image. The path itself is not used.
		imgPath = append(imgPath, "clipboard")
	}

	if len(imgPath) == 0 {
		log.Fatal("No images provided.")
	}

	var img []imageW.TranslatorImage

	for _, paths := range imgPath {
		log.Debugf("Getting image info for: %v", imgPath)
		newImage := imageW.Open(paths, *urlImagePtr, *clipImagePtr)
		img = append(img, newImage)
	}

	// We need this ratio to scale the image down/up to the required starting size.
	ratio := imageW.GetRatio(img[0].Dimensions, maxDim)
	firstWidth := float32(img[0].Dimensions.Width)
	firstHeight := float32(img[0].Dimensions.Height)

	go func() {
		// Create new window.
		w := app.NewWindow(
			app.Title("Manga Translator"),

			app.Size(unit.Dp(ratio*firstWidth), unit.Dp(ratio*firstHeight)),
			app.MinSize(unit.Dp(600), unit.Dp(300)),
		)

		if err := window.DrawFrame(w, img, cfg); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
