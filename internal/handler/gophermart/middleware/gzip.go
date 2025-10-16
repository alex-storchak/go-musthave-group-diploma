package middleware

import (
	"compress/gzip"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
		c.w.Header().Del("Content-Length")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func supportsCompression(r *http.Request, method string) bool {
	return searchEncodingMethod(r.Header.Get("Accept-Encoding"), method)
}

func isCompression(r *http.Request, method string) bool {
	return searchEncodingMethod(r.Header.Get("Content-Encoding"), method)
}

func searchEncodingMethod(methodStr string, method string) bool {
	if methodStr == "" {
		return false
	}

	encodings := strings.Split(methodStr, ",")
	for _, enc := range encodings {
		enc = strings.TrimSpace(enc)
		if enc == method {
			return true
		}
	}
	return false
}

func GzipMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ow := w

			if supportsCompression(r, "gzip") {
				cw := newCompressWriter(w)
				ow = cw

				defer cw.Close()
			}

			if isCompression(r, "gzip") {
				cr, err := newCompressReader(r.Body)
				if err != nil {
					logger.Error("decompression error", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				r.Body = cr
				defer cr.Close()
			}

			next.ServeHTTP(ow, r)
		})
	}
}
