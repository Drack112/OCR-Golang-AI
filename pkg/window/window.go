package window

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	gclip "gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Drack112/Anime-OCR-Translator/pkg/config"
	"github.com/Drack112/Anime-OCR-Translator/pkg/detect"
	imageW "github.com/Drack112/Anime-OCR-Translator/pkg/image"
	"github.com/gonoto/notosans"
	log "github.com/sirupsen/logrus"
)

type (
	D = layout.Dimensions
	C = layout.Context
)

var (
	DarkGray  = color.NRGBA{R: 0x2B, G: 0x2B, B: 0x2B, A: 0xFF}
	Gray      = color.NRGBA{R: 0x75, G: 0x75, B: 0x75, A: 0xFF}
	LightGray = color.NRGBA{R: 0xCF, G: 0xCF, B: 0xCF, A: 0xFF}
)

var preLoadPages = 2

func DrawFrame(w *app.Window, images []imageW.TranslatorImage, cfg config.File) error {

	var ops op.Ops

	var split = VSplit{Ratio: 0.60}

	var p pageList
	p.add(images)

	log.Debugf("Number of pages loaded: %d", p.len)

	go p.pages[p.idx].load(w, &cfg)
	p.preLoad(preLoadPages, w, &cfg)

	var (
		originalBtn   = new(widget.Clickable)
		translatedBtn = new(widget.Clickable)
	)

	fonts := gofont.Collection()
	fonts = appendOTC(fonts, text.Font{Typeface: "Noto"}, notosans.OTC())
	th := material.NewTheme(fonts)

	var (
		selectedO string // Original text
		selectedT string // Translated text
	)

	for {
		select {
		case e := <-w.Events():
			switch e := e.(type) {

			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				for i, b := range p.pages[p.idx].blocks {
					if p.pages[p.idx].blockButtons[i].Clicked() {
						log.Debugf("Clicked Block %d", i)
						selectedO = b.Text
						selectedT = b.Translated
					}
				}

				if originalBtn.Clicked() {
					if !p.pages[p.idx].text.finished || !p.pages[p.idx].text.ok {
						// Loading or error status.
						w.WriteClipboard(p.pages[p.idx].text.status)
					} else {
						// Original text. Detection and translation completed and succeeded.
						w.WriteClipboard(selectedO)
					}
				} else if translatedBtn.Clicked() {
					w.WriteClipboard(selectedT)
				}

				// Background
				layout.Center.Layout(gtx, func(gtx C) D {
					return colorBox(gtx, gtx.Constraints.Max, DarkGray)
				})

				// Application
				split.Layout(gtx, func(gtx C) D {
					return imageWidget(gtx, th, p)
				}, func(gtx C) D {
					return translatorPanelWidget(gtx, th, p.pages[p.idx].text, originalBtn, translatedBtn, p.pages[p.idx].text.status, selectedO, selectedT)
				})
				e.Frame(gtx.Ops)

			case key.Event:
				if e.State == key.Press {
					if (e.Name == "→" || e.Name == "D") && p.idx < p.len-1 {
						p.idx++
						selectedO, selectedT = "", ""
						p.preLoad(preLoadPages, w, &cfg)
						w.Invalidate()
					} else if (e.Name == "←" || e.Name == "A") && p.idx > 0 {
						p.idx--
						selectedO, selectedT = "", ""
						w.Invalidate()
					}
				}

			case system.DestroyEvent:
				return e.Err
			}
		}
	}
}

type pageList struct {
	pages []page
	idx   int // Current page.
	len   int
}

func (p *pageList) add(images []imageW.TranslatorImage) {
	for _, img := range images {
		newPage := page{
			image: img,
		}
		p.pages = append(p.pages, newPage)
		p.len++
	}
}

func (p *pageList) preLoad(num int, w *app.Window, cfg *config.File) {
	for i := 1; i <= num && i+p.idx < p.len; i++ {
		go p.pages[i+p.idx].load(w, cfg)
	}
}

type page struct {
	image        imageW.TranslatorImage
	blocks       []detect.TextBlock
	blockButtons []widget.Clickable
	text         textBlocks
}

func (p *page) load(w *app.Window, cfg *config.File) {
	if !p.text.loading && !p.text.finished {
		p.text.getText(w, cfg, p.image, &p.blocks, &p.blockButtons)
	}
}

func imageWidget(gtx C, th *material.Theme, p pageList) D {
	mainImg := layout.Center.Layout(gtx, func(gtx C) D {
		imgWidget := widget.Image{
			Fit:      widget.Contain,
			Position: layout.Center,
			Src:      paint.NewImageOp(p.pages[p.idx].image.Image),
		}.Layout(gtx)

		var blockWidgets []layout.StackChild

		if p.pages[p.idx].text.finished {
			for i, block := range p.pages[p.idx].blocks {
				blockWidgets = append(blockWidgets, blockBox(imgWidget, p.pages[p.idx].image.Dimensions, block, &p.pages[p.idx].blockButtons[i]))
			}
		}

		layout.Stack{}.Layout(gtx, blockWidgets...)

		return imgWidget
	},
	)
	if p.len > 1 {
		pageNum := fmt.Sprintf("%d/%d", p.idx+1, p.len)
		return layout.NW.Layout(gtx, func(gtx C) D {
			return layout.Inset{
				Left: unit.Dp(4),
				Top:  unit.Dp(4),
			}.Layout(gtx, func(gtx C) D {
				pageLabel := material.Label(th, unit.Dp(20), pageNum)
				pageLabel.Color = LightGray
				return pageLabel.Layout(gtx)
			})
		})
	} else {
		return mainImg
	}
}

func translatorPanelWidget(gtx C, th *material.Theme, txt textBlocks, originalBtn, translatedBtn *widget.Clickable, status, selectedO, selectedT string) D {
	if !txt.finished {
		return translatorWidget(gtx, th, originalBtn, status, "Loading...")
	} else if !txt.ok {
		return translatorWidget(gtx, th, originalBtn, status, "Error")
	} else {
		var tlSplit HSplit

		return tlSplit.Layout(gtx, func(gtx C) D {
			return translatorWidget(gtx, th, originalBtn, selectedO, "Original Text")
		}, func(gtx C) D {
			return translatorWidget(gtx, th, translatedBtn, selectedT, "Translated Text")
		})
	}
}

func colorBox(gtx C, size image.Point, color color.NRGBA) D {
	area := gclip.Rect{Max: size}.Push(gtx.Ops)
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	area.Pop()
	return D{Size: size}
}

func appendOTC(collection []text.FontFace, fnt text.Font, otc []byte) []text.FontFace {
	face, err := opentype.ParseCollection(otc)
	if err != nil {
		log.Fatalf("Failed to parse font collection: %v", err)
	}
	return append(collection, text.FontFace{Font: fnt, Face: face})
}
