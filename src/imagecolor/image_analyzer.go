package imagecolor

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"limiter"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)
import _ "image/jpeg"
import _ "image/png"
import _ "image/gif"

var logger = log.New(os.Stdout, "image:", log.Lshortfile)

var (
	ResultWriteError       = errors.New("can't write results - just stop")
	UnsupportedFormatError = errors.New("unsupported format")
)

type imageAnalyzer struct {
	queue   *Feeder
	memlim  *limiter.Memory
	maxAnal *limiter.Limiter // :)
	wg      sync.WaitGroup

	results     io.Writer
	resultsLock sync.Mutex
}

func NewImageAnalyzer(queue *Feeder, memlim *limiter.Memory, results io.Writer) *imageAnalyzer {
	return &imageAnalyzer{
		queue:       queue,
		memlim:      memlim,
		maxAnal:     limiter.New(maxParallelAnalysis),
		results:     results,
		resultsLock: sync.Mutex{},
		wg:          sync.WaitGroup{},
	}
}

func (ia *imageAnalyzer) Run() {
	for {
		item := ia.queue.Get()
		if item == nil {
			break
		}

		ia.maxAnal.Add(1)
		ia.wg.Add(1)
		go ia.processImage(item)
	}

	ia.wg.Wait()
}

func (ia *imageAnalyzer) processImage(item *QueueItem) {
	start := time.Now()

	defer ia.wg.Done()
	defer ia.maxAnal.Sub(1)
	defer item.Clean()
	defer func() {
		elapsed := time.Since(start)
		log.Printf("image %s proccesed in %s", item.Name, elapsed)
	}()

	data := item.GetData()
	if data == nil {
		fmt.Println("Data are empty")
		return
	}

	ia.memlim.CheckAddition(placeholderFileSize) // poors man solution but faster

	img, name, err := image.Decode(data)
	ia.memlim.CheckRelease()
	if err != nil {
		logger.Println(err, item.Name)
		return
	}
	if name != "jpeg" && name != "png" && name != "gif" {
		logger.Println(UnsupportedFormatError)
		return
	}

	colors := ia.processRectSort(img)
	ia.resultsLock.Lock()
	defer ia.resultsLock.Unlock()
	_, err = ia.results.Write([]byte(item.Name))
	if err != nil {
		panic(ResultWriteError)
	}
	for _, c := range colors {
		_, err = ia.results.Write([]byte(" " + c.ToHex()))
		if err != nil {
			panic(ResultWriteError)
		}
	}
	_, err = ia.results.Write([]byte("\n"))
	if err != nil {
		panic(ResultWriteError)
	}

	return
}

type Result struct {
	Color color.Color
	Count int64
}

func (res *Result) ToHex() string {
	r, g, b, _ := res.Color.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", int(r>>8), int(g>>8), int(b>>8))
}

func (ia *imageAnalyzer) processRectBrute(r image.Image) []Result {
	result := make([]Result, 3)

	x := r.Bounds().Dx()
	y := r.Bounds().Dy()
	colors := make(map[color.Color]int64)
	for i := r.Bounds().Min.X; i < x; i++ {
		for ii := r.Bounds().Min.Y; ii < y; ii++ {
			c := r.At(i, ii)
			colors[c]++
			if result[2].Count < colors[c] {
				if result[1].Count < colors[c] {
					if result[0].Count < colors[c] {
						result[0].Count = colors[c]
						result[0].Color = c
					} else {
						result[1].Count = colors[c]
						result[1].Color = c
					}
				} else {
					result[2].Count = colors[c]
					result[2].Color = c
				}
			}
		}
	}

	return result
}

func (ia *imageAnalyzer) processRectSort(r image.Image) []Result {
	x := r.Bounds().Dx()
	y := r.Bounds().Dy()
	colors := make(map[color.Color]int64)
	for i := r.Bounds().Min.X; i < x; i++ {
		for ii := r.Bounds().Min.Y; ii < y; ii++ {
			colors[r.At(i, ii)]++
		}
	}

	var ss []Result
	for k, v := range colors {
		ss = append(ss, Result{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Count > ss[j].Count
	})

	if len(ss) > 3 {
		return ss[0:3]
	}
	return ss
}
