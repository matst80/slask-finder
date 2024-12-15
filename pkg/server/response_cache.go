package server

//type cacheWriter struct {
//	key      string
//	duration time.Duration
//	store    func(string, []byte, time.Duration) error
//}
//
//func (cw *cacheWriter) Write(p []byte) (n int, err error) {
//	err = cw.store(cw.key, p, cw.duration)
//	return len(p), err
//}
//
//func MakeCacheWriter(w io.Writer, key string, setRaw func(string, []byte, time.Duration) error) io.Writer {
//
//	cacheWriter := &cacheWriter{
//		key:      key,
//		duration: time.Second * (60 * 5),
//		store:    setRaw,
//	}
//
//	return io.MultiWriter(w, cacheWriter)
//
//}
