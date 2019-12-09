package imagecolor

import (
	"limiter"
	"os"
	"testing"
)

func TestImageAnalyzer_Run(t *testing.T) {
	list, err := os.Open("../../data/image_list")
	if err != nil {
		t.Fatal(err)
	}
	ml := limiter.NewMemory(uint64(512 * MB))
	f := NewFeeder(list, ml, "../../data/cache")
	f.Run()

	res, err := os.Create("../../data/results")
	if err != nil {
		t.Fatal(err)
	}

	ia := NewImageAnalyzer(f, ml, res)
	ia.Run()
}
