package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/jdeng/goheif"
	"github.com/rs/zerolog"
)

type WebService struct {
	log      zerolog.Logger
	r        *mux.Router
	fs       http.Handler
	maxBytes int64
	cnt      *uint64
	act      chan struct{}
}

func newWebService(log zerolog.Logger, conf *Config) (*WebService, error) {
	ws := new(WebService)
	ws.log = log.With().Str("component", "webservice").Logger()
	ws.r = mux.NewRouter()
	ws.fs = http.FileServer(http.Dir(conf.ServePath))

	ws.maxBytes = conf.MaxSizeMB << 20
	ws.cnt = new(uint64)
	*ws.cnt = 0
	ws.act = make(chan struct{}, conf.MaxConcurrent)
	for i := 0; i < conf.MaxConcurrent; i++ {
		ws.act <- struct{}{}
	}

	// form page
	ws.r.Methods(http.MethodGet).Path("/").HandlerFunc(ws.serveFiles)
	ws.r.Methods(http.MethodPost).Path("/convert").HandlerFunc(ws.postConvert)

	// assets
	//ws.r.Methods(http.MethodGet).PathPrefix("/js/").HandlerFunc(ws.serveFiles)
	ws.r.Methods(http.MethodGet).PathPrefix("/css/").HandlerFunc(ws.serveFiles)
	ws.r.Methods(http.MethodGet).PathPrefix("/fonts/").HandlerFunc(ws.serveFiles)

	// internal stuff
	ws.r.Methods(http.MethodGet).Path("/check").HandlerFunc(ws.getCheck)
	ws.r.Methods(http.MethodGet).Path("/count").HandlerFunc(ws.getCount)

	return ws, nil
}

func (ws *WebService) serveFiles(w http.ResponseWriter, r *http.Request) {
	ws.logRequest(r)
	ws.fs.ServeHTTP(w, r)
}

func (ws *WebService) getCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Probably fine."))
}

func (ws *WebService) getCount(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(strconv.FormatUint(atomic.LoadUint64(ws.cnt), 10)))
}

func (ws *WebService) postConvert(w http.ResponseWriter, r *http.Request) {
	// todo: break this func up
	// todo: handle exif data

	ws.logRequest(r)

	// always close your jams
	defer func() {
		_, _ = io.Copy(ioutil.Discard, r.Body)
		_ = r.Body.Close()
	}()

	var (
		act        struct{}
		imageBytes []byte
		infileName string
		outname    string
	)

	select {
	// wait for max of 2 seconds
	// todo: limit actual maximum # of connections?
	case <-time.After(2 * time.Second):
		ws.log.Warn().Msg("Took too long to acquire action")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Too many concurrent requests, try again later"))
		return

	// in case request is cancelled
	case <-r.Context().Done():
		ws.log.Error().Err(r.Context().Err()).Msg("Request context expired")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte(fmt.Sprintf("Request timeout: %v", r.Context().Err())))
		return

	// retrieve action ticket
	case act = <-ws.act:
		defer func() { ws.act <- act }()
	}

	// fetch multipart reader
	mpr, err := r.MultipartReader()
	if err != nil {
		ws.log.Error().Err(err).Msg("Error creating multipart reader")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("error reading body: %v", err)))
		return
	}

	// parse out parts
	for {
		part, err := mpr.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			ws.log.Error().Err(err).Msg("Error reading form data")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(fmt.Sprintf("Error reading body: %v", err)))
			return
		}
		switch part.FormName() {
		// create input file reader
		case "infile":
			infileName = part.FileName()
			mbr := http.MaxBytesReader(w, part, ws.maxBytes)
			imageBytes, err = ioutil.ReadAll(mbr)
			_ = mbr.Close()
			if err != nil {
				ws.log.Error().Err(err).Msg("Error reading image data")
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(fmt.Sprintf("Error reading image data: %v", err)))
				return
			}

		// limit output name to 512 bytes
		case "outname":
			mbr := http.MaxBytesReader(w, part, 512)
			b, err := ioutil.ReadAll(mbr)
			_ = mbr.Close()
			if err != nil {
				ws.log.Error().Err(err).Msg("Error reading outname")
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(fmt.Sprintf("Error reading outname: %v", err)))
				return
			}
			outname = string(b)

		case "submit":
			// do nothing

		default:
			ws.log.Warn().Msgf("Unexpected form field %q seen", part.FormName())
		}
	}

	img, err := goheif.Decode(bytes.NewBuffer(imageBytes))
	if err != nil {
		ws.log.Error().Err(err).Msg("Error decoding file")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(fmt.Sprintf("error decoding file: %v", err)))
		return
	}

	rdr := bytes.NewReader(imageBytes)

	exif, err := goheif.ExtractExif(rdr)
	if err != nil {
		ws.log.Warn().Err(err).Msg("No EXIF data found")
	}

	if outname == "" {
		outname = fmt.Sprintf("%s.jpg", path.Base(infileName))
	}

	buff := bytes.NewBuffer(nil)

	iw, err := newWriterExif(buff, exif)
	if err != nil {
		ws.log.Error().Err(err).Msg("Error writing EXIF data")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("Error writing EXIF data: %v", err)))
		return
	}

	if err = jpeg.Encode(iw, img, nil); err != nil {
		ws.log.Error().Err(err).Msg("Error encoding jpeg")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("Error encoding to jpeg: %v", err)))
		return
	}

	atomic.AddUint64(ws.cnt, 1)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", outname))
	w.Header().Set("Content-Length", strconv.Itoa(buff.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buff.Bytes())
}

func (ws *WebService) logRequest(req *http.Request) {
	ws.log.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Str("requestURI", req.RequestURI).
		Interface("headers", req.Header).
		Msg("Incoming request")
}

func (ws *WebService) serve(ip string, port int) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	ws.log.Info().Msgf("Listener up: %s", addr)
	return http.ListenAndServe(addr, ws.r)
}
