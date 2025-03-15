package window

import (
	"errors"
	"image"
	"math"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	gclip "gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/Drack112/Anime-OCR-Translator/pkg/cache"
	"github.com/Drack112/Anime-OCR-Translator/pkg/config"
	"github.com/Drack112/Anime-OCR-Translator/pkg/detect"
	imageW "github.com/Drack112/Anime-OCR-Translator/pkg/image"
	"github.com/Drack112/Anime-OCR-Translator/pkg/translate"
	"github.com/labstack/gommon/log"
)

type textBlocks struct {
	status   string // Loading status.
	loading  bool   // Is true if the process is in progress.
	finished bool   // Is true the process is complete.
	ok       bool   // Is true the process did not encounter any errors.
}

func (t *textBlocks) getText(w *app.Window, cfg *config.File, img imageW.TranslatorImage, blocks *[]detect.TextBlock, blockButtons *[]widget.Clickable) {
	t.loading = true

	defer func() {
		t.loading = false
		t.finished = true
		t.ok = t.status == `Done!`
		w.Invalidate()
	}()

	var blankCfg config.File

	if *cfg == blankCfg {
		t.status = `Your config is either blank or doesn't exist, run the "manga-translator-setup" application to create one.`
		return
	}
	var translateOnly bool

	*blocks, translateOnly = cache.Check(img.Hash, cfg.Translation.SelectedService)

	if *blocks == nil || translateOnly {
		var err error
		if !translateOnly {
			t.status = `Detecting text...`

			annotation, err := detect.GetAnnotation(img.Image)
			if err != nil {
				*blocks = []detect.TextBlock{}
				t.status = err.Error()
				return
			}

			*blocks = detect.OrganizeAnnotation(annotation)
		}

		var allOriginal []string
		for _, block := range *blocks {
			*blockButtons = append(*blockButtons, widget.Clickable{})
			allOriginal = append(allOriginal, block.Text)
		}

		t.status = `Translating text...`
		log.Infof("Translating detected text with: %v", cfg.Translation.SelectedService)

		var allTranslated []string
		if cfg.Translation.SelectedService == "google" {
			allTranslated, err = translate.GoogleTranslate(
				allOriginal,
				cfg.Translation.SourceLanguage,
				cfg.Translation.TargetLanguage,
				cfg.Translation.Google.APIKey,
			)
		} else if cfg.Translation.SelectedService == "deepL" {
			allTranslated, err = translate.DeepLTranslate(
				allOriginal,
				cfg.Translation.SourceLanguage,
				cfg.Translation.TargetLanguage,
				cfg.Translation.DeepL.APIKey,
			)
		} else {
			t.status = `Your config does not have a valid selected service, run the "manga-translator-setup" application again.`
			err = errors.New("no selected service")
		}
		for i, txt := range allTranslated {
			(*blocks)[i].Translated = txt
		}
		if err == nil {
			cache.Add(img.Hash, cfg.Translation.SelectedService, *blocks)
		} else {
			t.status = allTranslated[0]
			return
		}
	} else {

		for range *blocks {
			*blockButtons = append(*blockButtons, widget.Clickable{})
		}
	}
	t.status = `Done!`
}

func blockBox(img D, originalDims imageW.Dimensions, block detect.TextBlock, btn *widget.Clickable) layout.StackChild {
	return layout.Stacked(
		func(gtx C) D {

			ratio := imageW.GetRatio(originalDims, float32(math.Max(float64(img.Size.X), float64(img.Size.Y))))

			op.Offset(
				f32.Pt(float32(block.Vertices[0].X)*ratio,
					float32(block.Vertices[0].Y)*ratio),
			).Add(gtx.Ops)

			boxSizeX := float32(block.Vertices[1].X - block.Vertices[0].X)
			boxSizeY := float32(block.Vertices[2].Y - block.Vertices[1].Y)
			gtx.Constraints.Max = image.Point{
				X: int(boxSizeX * ratio),
				Y: int(boxSizeY * ratio),
			}

			box := func(gtx C) D {
				area := gclip.Rect{
					Max: image.Point{
						X: int(boxSizeX * ratio),
						Y: int(boxSizeY * ratio),
					},
				}.Push(gtx.Ops)

				fillColor := block.Color
				fillColor.A = 0x40
				paint.ColorOp{Color: fillColor}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				defer area.Pop()
				return D{Size: gtx.Constraints.Max}
			}

			borderedBox := func(gtx C) D {
				return widget.Border{
					Color:        block.Color,
					CornerRadius: unit.Dp(1),
					Width:        unit.Dp(2),
				}.Layout(gtx, box)
			}

			return Clickable(gtx, btn, true, borderedBox)
		},
	)
}
