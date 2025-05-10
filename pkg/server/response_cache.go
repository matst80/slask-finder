package server

//type cacheWriter struct {
//	serverApiKey      string
//	duration time.Duration
//	store    func(string, []byte, time.Duration) error
//}
//
//func (cw *cacheWriter) Write(p []byte) (n int, err error) {
//	err = cw.store(cw.serverApiKey, p, cw.duration)
//	return len(p), err
//}
//
//func MakeCacheWriter(w io.Writer, serverApiKey string, setRaw func(string, []byte, time.Duration) error) io.Writer {
//
//	cacheWriter := &cacheWriter{
//		serverApiKey:      serverApiKey,
//		duration: time.Second * (60 * 5),
//		store:    setRaw,
//	}
//
//	return io.MultiWriter(w, cacheWriter)
//
//}
