package roaring

// EachContainer invokes fn once per container in ascending key order, where key
// is the high 16 bits shared by the container's values (value>>16) and bm is a
// freshly cloned single-container bitmap of those values. The clone is bulk, so
// the walk is O(containers), not O(cardinality). Return false to stop.
func (rb *Bitmap) EachContainer(fn func(key uint16, bm *Bitmap) bool) {
	ra := &rb.highlowcontainer
	for i, n := 0, ra.size(); i < n; i++ {
		sub := &Bitmap{}
		sub.highlowcontainer.appendContainer(ra.getKeyAtIndex(i), ra.getContainerAtIndex(i).clone(), false)
		if !fn(ra.getKeyAtIndex(i), sub) {
			return
		}
	}
}
