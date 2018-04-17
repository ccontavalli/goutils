package misc

import (
	"runtime"
)

// Computes the memory used by the supplied functions.
// Returns the memory that the function caused to be allocated, followed
// by the number of allocs and frees performed.
func CollectMemoryStats(function func()) (memory int64, mallocs uint64, frees uint64) {
	var b, a runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&b)
	function()
	runtime.GC()
	runtime.ReadMemStats(&a)

	return int64(a.HeapAlloc) - int64(b.HeapAlloc), a.Mallocs - b.Mallocs, a.Frees - b.Frees
}
