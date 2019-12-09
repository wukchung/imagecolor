package main

import (
	"flag"
	"fmt"
	"imagecolor"
	"limiter"
	"log"
	"os"
)

func main() {
	log.SetOutput(os.Stdout)

	source := flag.String("source", "", "filename of items to process")
	results := flag.String("results", "", "filename to save results to")
	cacheDir := flag.String("cache", "", "cache directory")
	memlim := flag.Int64("memory", 512, "memory limit in MB")

	flag.Parse()

	if *source == "" || *results == "" {
		fmt.Println("usage: -source=/path/to/source/file -results=/path/to/result/file -cache=/path/to/result/file -memory=512")
		os.Exit(1)
	}

	list, err := os.Open(*source)
	if err != nil {
		panic(err)
	}
	ml := limiter.NewMemory(uint64(*memlim * imagecolor.MB))
	f := imagecolor.NewFeeder(list, ml, *cacheDir)
	f.Run()

	res, err := os.Create(*results)
	if err != nil {
		panic(err)
	}

	ia := imagecolor.NewImageAnalyzer(f, ml, res)
	ia.Run()
}
