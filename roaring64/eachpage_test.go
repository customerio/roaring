package roaring64

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pageSize = uint64(1) << 16

// naiveSplit reproduces EachPage's grouping by iterating every set bit, as an
// independent oracle.
func naiveSplit(bm *Bitmap) (pages []uint64, bms []*Bitmap) {
	var (
		curPage uint64
		curBM   *Bitmap
	)
	it := bm.Iterator()
	for it.HasNext() {
		v := it.Next()
		page := v >> 16
		if curBM == nil || page != curPage {
			if curBM != nil {
				pages = append(pages, curPage)
				bms = append(bms, curBM)
			}
			curPage = page
			curBM = New()
		}
		curBM.Add(v)
	}
	if curBM != nil {
		pages = append(pages, curPage)
		bms = append(bms, curBM)
	}
	return pages, bms
}

func collectPages(bm *Bitmap) (pages []uint64, bms []*Bitmap) {
	bm.EachPage(func(page uint64, sub *Bitmap) bool {
		pages = append(pages, page)
		bms = append(bms, sub)
		return true
	})
	return pages, bms
}

func assertEachPageMatches(t *testing.T, vals ...uint64) {
	t.Helper()
	bm := BitmapOf(vals...)
	wantPages, wantBMs := naiveSplit(bm)
	gotPages, gotBMs := collectPages(bm)

	require.Equal(t, wantPages, gotPages)
	require.Len(t, gotBMs, len(wantBMs))
	union := New()
	for i := range gotBMs {
		assert.Truef(t, wantBMs[i].Equals(gotBMs[i]), "page %d mismatch", gotPages[i])
		it := gotBMs[i].Iterator()
		for it.HasNext() {
			require.Equal(t, gotPages[i], it.Next()>>16, "value on wrong page")
		}
		union.Or(gotBMs[i])
	}
	require.True(t, union.Equals(bm), "pages must reassemble the input")
}

func TestEachPage_MatchesNaive(t *testing.T) {
	cases := map[string][]uint64{
		"empty":         {},
		"single":        {42},
		"one page":      {0, 1, pageSize - 1},
		"boundary":      {pageSize - 1, pageSize, pageSize + 1},
		"sparse":        {5, pageSize + 5, 7 * pageSize, 100 * pageSize},
		"above 2^32":    {5, 1 << 32, (1 << 32) + pageSize + 7, (3 << 32) | (5 * pageSize) | 7},
		"straddle 2^32": {pageSize - 1, pageSize, (1 << 32) - 1, 1 << 32, (1 << 32) + 1},
	}
	for name, vals := range cases {
		vals := vals
		t.Run(name, func(t *testing.T) { assertEachPageMatches(t, vals...) })
	}
}

func TestEachPage_Random(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for iter := 0; iter < 200; iter++ {
		bm := New()
		n := rng.Intn(3000)
		for j := 0; j < n; j++ {
			switch rng.Intn(4) {
			case 0:
				bm.Add(uint64(rng.Intn(int(3 * pageSize))))
			case 1:
				bm.Add(uint64(rng.Intn(500))*pageSize + uint64(rng.Intn(int(pageSize))))
			case 2:
				bm.Add(uint64(rng.Intn(500)) * pageSize)
			default:
				bm.Add((uint64(rng.Intn(4)) << 32) | uint64(rng.Int63n(int64(10*pageSize))))
			}
		}
		wantPages, wantBMs := naiveSplit(bm)
		gotPages, gotBMs := collectPages(bm)
		require.Equal(t, wantPages, gotPages)
		require.Len(t, gotBMs, len(wantBMs))
		for i := range gotBMs {
			require.Truef(t, wantBMs[i].Equals(gotBMs[i]), "iter %d page %d mismatch", iter, gotPages[i])
		}
	}
}

// TestEachPage_ClonesContainers guards that returned page bitmaps are independent
// copies: callers mutate them, so they must not alias the source's containers.
func TestEachPage_ClonesContainers(t *testing.T) {
	bm := BitmapOf(1, 2, pageSize+1)
	_, bms := collectPages(bm)
	require.NotEmpty(t, bms)
	bms[0].Add(999999)
	require.False(t, bm.Contains(999999), "mutating a page bitmap must not touch the source")
	bm.Add(3)
	require.False(t, bms[0].Contains(3), "mutating the source must not touch a returned page bitmap")
}
