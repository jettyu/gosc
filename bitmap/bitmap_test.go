package bitmap_test

import (
	"testing"

	"github.com/jettyu/gosc/bitmap"
)

// TestBitmap ...
func TestBitmap(t *testing.T) {
	testSet := func(bm *bitmap.Bitmap, adds []uint, t *testing.T) {
		for _, v := range adds {
			bm.Set(v)
		}
		for _, v := range adds {
			if !bm.Has(v) {
				t.Fatal(v)
			}
		}
	}
	testDel := func(bm *bitmap.Bitmap, dels []uint, t *testing.T) {
		for _, v := range dels {
			bm.Del(v)
		}
		for _, v := range dels {
			if bm.Has(v) {
				t.Fatal(v)
			}
		}
	}
	bm := bitmap.NewBitmap(64)
	adds := []uint{2, 3, 10, 15, 60}
	dels := []uint{1, 3, 7, 15}
	testSet(bm, adds, t)
	testDel(bm, dels, t)
	testSet(bm, adds, t)
	bm.ResetMax(128)
	adds1 := []uint{100, 102, 103, 108}
	testSet(bm, adds1, t)
	for _, v := range append(adds, adds1...) {
		if !bm.Has(v) {
			t.Fatal(v)
		}
	}
}
