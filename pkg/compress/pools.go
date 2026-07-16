package compress

import "sync"

const readBufferSize = 256 * 1024

var readBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, readBufferSize)
		return &buf
	},
}

func getReadBuffer() []byte {
	return *readBufferPool.Get().(*[]byte)
}

func putReadBuffer(buf []byte) {
	readBufferPool.Put(&buf)
}
