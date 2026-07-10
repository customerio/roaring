package roaring64

import "github.com/RoaringBitmap/roaring/v2"

// EachPage invokes fn once per non-empty 2^16-aligned page (page = value>>16) in
// ascending order, passing a freshly cloned single-page bitmap. A page maps 1:1 to
// a roaring container, so this is O(pages), not O(cardinality). Return false to stop.
func (rb *Bitmap) EachPage(fn func(page uint64, bm *Bitmap) bool) {
	ra := &rb.highlowcontainer
	for i, n := 0, ra.size(); i < n; i++ {
		hi := uint64(ra.getKeyAtIndex(i)) // the shared value>>32 for this container
		stopped := false
		ra.getContainerAtIndex(i).EachContainer(func(key uint16, sub *roaring.Bitmap) bool {
			page := hi<<16 | uint64(key)
			wrapped := &Bitmap{}
			wrapped.highlowcontainer.appendContainer(uint32(hi), sub, false)
			if !fn(page, wrapped) {
				stopped = true
				return false
			}
			return true
		})
		if stopped {
			return
		}
	}
}
