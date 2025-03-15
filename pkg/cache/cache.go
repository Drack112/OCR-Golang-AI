package cache

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/Drack112/Anime-OCR-Translator/pkg/config"
	"github.com/Drack112/Anime-OCR-Translator/pkg/detect"
)

type data struct {
	Hash    string
	Service string
	Blocks  []detect.TextBlock
}

var mu sync.Mutex

func read() []data {
	var cacheData []data

	cachePath := filepath.Join(config.Path(), "mtl-cache.bin")
	cacheFile, err := os.Open(cachePath)
	if errors.Is(err, os.ErrNotExist) {
		cacheFile, err = os.Create(cachePath)
		if err != nil {
			log.Fatal(err)
		}
		defer cacheFile.Close()

		enc := gob.NewEncoder(cacheFile)
		if err := enc.Encode(cacheData); err != nil {
			log.Fatalf("Cache creation failed: %v", err)
		}

		return cacheData
	} else if err != nil {
		log.Fatal(err)
	}
	defer cacheFile.Close()

	dec := gob.NewDecoder(cacheFile)
	if err := dec.Decode(&cacheData); err != nil {
		log.Fatal(err)
	}
	return cacheData
}

func Check(h string, service string) (blocks []detect.TextBlock, translateOnly bool) {
	mu.Lock()
	defer mu.Unlock()

	cacheData := read()
	var existingBlocks []detect.TextBlock
	for _, data := range cacheData {
		if h == data.Hash && data.Service == service {
			log.Info("Image found in cache, skipping API requests.")
			return data.Blocks, false
		} else if h == data.Hash {
			existingBlocks = data.Blocks
		}
	}

	if existingBlocks != nil {
		log.Info("Image text found in cache, performing API requests")
		return existingBlocks, true
	}

	log.Info("Image not found in cache, performing API requests")
	return nil, false
}

func Add(h string, service string, blocks []detect.TextBlock) {
	mu.Lock()
	defer mu.Unlock()

	log.Debugf("Adding new image to cache. sha256:%v", h)
	cacheData := read()

	cachePath := filepath.Join(config.Path(), "mtl-cache.bin")
	cacheFile, err := os.OpenFile(cachePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer cacheFile.Close()

	newData := data{
		Hash:    h,
		Service: service,
		Blocks:  blocks,
	}
	cacheData = append(cacheData, newData)
	enc := gob.NewEncoder(cacheFile)
	if err := enc.Encode(cacheData); err != nil {
		log.Fatalf("Cache write failed: %v", err)
	}
}
