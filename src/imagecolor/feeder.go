package imagecolor

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"limiter"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type QueueItem struct {
	Size     int64
	Filename string
	Name     string
	memlim   *limiter.Memory
	Data     io.ReadCloser
	file     *os.File
}

func (qi *QueueItem) GetData() io.ReadCloser {
	if qi.Data != nil {
		return qi.Data
	}

	if qi.Filename != "" {
		f, err := os.Open(qi.Filename)
		if err != nil {
			log.Println(err)
			return nil
		}
		qi.file = f
		return qi.file
	}
	return nil
}

func (qi *QueueItem) Clean() {
	if qi.Data != nil {
		qi.Data.Close()
	}

	if qi.file != nil {
		err := qi.file.Close()
		if err != nil {
			log.Println(err)
		}
		qi.file = nil
		log.Println("Closing open file", qi.Filename)
	}

	if qi.Filename != "" {
		err := os.Remove(qi.Filename)
		if err != nil {
			go func() { // I hope it's not data race condition in my code but it seems on Windows there is some delay when OS release a lock
				time.Sleep(time.Second)
				err := os.Remove(qi.Filename)
				if err != nil {
					log.Println(err)
				}
			}()
		}
	}

	qi.memlim.CheckRelease()
}

type Feeder struct {
	list     io.ReadCloser // list of images
	cacheDir string

	dataQueue      []*QueueItem // images loaded in memlim
	cacheDataQueue []*QueueItem // image saved to HDD

	maxDownloads *limiter.Limiter
	memlim       *limiter.Memory
	dlHistory    *history

	lock   *sync.Mutex
	wg     *sync.WaitGroup
	change chan struct{}
	done   chan struct{}
}

func NewFeeder(list io.ReadCloser, memory *limiter.Memory, cacheDir string) *Feeder {
	err := os.Mkdir(cacheDir, 0655)
	if err != nil && !os.IsExist(err) {
		fmt.Println(err)
	}

	return &Feeder{
		list:         list,
		memlim:       memory,
		maxDownloads: limiter.New(maxParallelDownloads),
		dlHistory: &history{
			lock:  &sync.Mutex{},
			items: map[string]struct{}{},
		},
		change:   make(chan struct{}),
		done:     make(chan struct{}),
		lock:     &sync.Mutex{},
		wg:       &sync.WaitGroup{},
		cacheDir: cacheDir,
	}
}

func (f *Feeder) Run() {
	go func() {
		rr := bufio.NewReader(f.list)
		defer f.list.Close()
		for {
			url, err := rr.ReadString('\n')
			if err == io.EOF {
				break
			}
			f.maxDownloads.Add(1)
			f.wg.Add(1)
			go func() {
				defer f.wg.Done()
				defer f.maxDownloads.Sub(1)
				defer func() {
					f.change <- struct{}{}
				}()
				url := strings.Trim(url, "\n")
				if !f.dlHistory.Add(url) {
					logger.Println("duplicate detected: ", url)
					return
				}
				resp, err := http.Get(url)
				if err != nil {
					logger.Println("failed to download", url)
					return
				}
				if resp.StatusCode != http.StatusOK {
					logger.Println("wrong status code", url, resp.StatusCode)
					return
				}

				f.add(&QueueItem{
					Name:   url,
					Data:   resp.Body,
					Size:   resp.ContentLength,
					memlim: f.memlim,
				})
			}()
		}
		f.wg.Wait()
		f.done <- struct{}{}
	}()
}

func (f *Feeder) Get() *QueueItem {
	for {
		if item := f.get(); item != nil {
			return item
		}
		select {
		case <-f.change:
			continue
		case <-f.done:
			return nil
		}
	}
}

func (f *Feeder) get() *QueueItem {
	f.lock.Lock()
	defer f.lock.Unlock()

	if len(f.dataQueue) > 0 {
		var item *QueueItem
		item, f.dataQueue = f.dataQueue[len(f.dataQueue)-1], f.dataQueue[:len(f.dataQueue)-1]
		return item
	}

	if len(f.cacheDataQueue) > 0 {
		var item *QueueItem
		item, f.cacheDataQueue = f.cacheDataQueue[len(f.cacheDataQueue)-1], f.cacheDataQueue[:len(f.cacheDataQueue)-1]
		return item
	}

	return nil
}

func (f *Feeder) add(item *QueueItem) {
	f.lock.Lock()
	defer f.lock.Unlock()
	// add task to Data queue that stays loaded in memlim
	if len(f.dataQueue) < maxDataQueueItems {
		f.dataQueue = append(f.dataQueue, item)
		//f.memlim.CheckAddition(uint64(item.Size)) // this is added too late - unfortunate
		return
	}

	tmpFile, err := ioutil.TempFile(f.cacheDir, "tmp.")
	if err != nil {
		panic("we rather stop completely")
	}

	written, err := io.Copy(tmpFile, item.GetData())
	if written != item.Size {
		logger.Println("Filesize written to cache doesn't match the Size of downloaded file")
	}

	// data are not used anymore we will ise the cached file
	item.Data.Close()
	item.Data = nil
	item.Filename = tmpFile.Name()

	f.cacheDataQueue = append(f.cacheDataQueue, item)

	if len(f.cacheDataQueue) < maxCacheDataQueueItems {
		//TODO: at this point we should stop downloading and wait until some images are processed
		// we dont' want to hit max open file descriptors or hdd limits
	}

	return
}

/*
The history is used as a duplicate detector. Since we should expect `input files with more than a billion URLs`
it has limited size. This component can be improved a lot by introducing some quick DB to remove the limits.
Improving this is not in a scope of a task.
*/
type history struct {
	items map[string]struct{}
	lock  *sync.Mutex
}

var md5hash = md5.New()

func (d *history) Add(item string) (added bool) {

	if !enableHistory {
		return true
	}

	d.lock.Lock()
	defer d.lock.Unlock()
	if _, ok := d.items[item]; ok {
		return false
	}

	if maxHistorySize <= len(d.items) {
		d.items = map[string]struct{}{}
	}
	d.items[item] = struct{}{}

	return true
}
