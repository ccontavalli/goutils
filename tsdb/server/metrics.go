package server

import (
	"encoding/json"
	"fmt"
	"github.com/ccontavalli/goutils/httpu"
	"github.com/ccontavalli/goutils/misc"
	"github.com/ccontavalli/goutils/tsdb"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type lockedSerie struct {
	lock   sync.RWMutex
	reader *tsdb.SerieReader
}

type MetricsServer struct {
	MaxEntriesPerReply int

	basepath string
	sr       map[string]*lockedSerie
}

func New(path string) (*MetricsServer, error) {
	sr := make(map[string]*lockedSerie)
	series := tsdb.GetSeries(path)
	for _, serie := range series {
		basename := filepath.Base(serie)
		sr[basename] = &lockedSerie{}
	}

	return &MetricsServer{1000, path, sr}, nil
}

func (ms *MetricsServer) Register(url string, mux *http.ServeMux) {
	mux.HandleFunc(path.Join(url, "list"), ms.List)
	mux.HandleFunc(path.Join(url, "get", "offset")+"/", ms.GetOffset)
	mux.HandleFunc(path.Join(url, "get", "range")+"/", ms.GetRange)
}

type getRangeRequest struct {
	// Offset from the end of the first entry to get.
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
	// Maximum number of entries to return.
	Entries int `json:"entries"`
}

func (ms *MetricsServer) GetRange(w http.ResponseWriter, r *http.Request) {
}

type GetOffsetRequest struct {
	// Offset from the end of the first entry to get.
	Start uint64 `json:"start"`
	// How many entries to retrieve.
	Entries int `json:"entries"`
}

type GetOffsetReply struct {
	// The request as interpreted by the server.
	// If Entires was too high, the server may have reduced it.
	// To check if you are at the end of the serie, you need to compare
	// the # of pointers returned against the entries the server was
	// willing to return.
	Request GetOffsetRequest `json:"request"`
	Point   []tsdb.Point     `json:"point"`
}

func (ms *MetricsServer) getSerieReader(handler string, w http.ResponseWriter, r *http.Request) *lockedSerie {
	path := path.Clean(r.URL.Path)
	tostrip := handler
	index := strings.Index(path, tostrip)
	if index < 0 {
		http.Error(w, "unknown path", http.StatusBadRequest)
		return nil
	}

	serie := path[index+len(tostrip):]
	sr, ok := ms.sr[serie]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown serie '%s'", serie), http.StatusBadRequest)
		return nil
	}

	if sr.reader == nil {
		sr.lock.Lock()
		sr.reader = tsdb.NewSerieReader(filepath.Join(ms.basepath, serie))
		err := sr.reader.Open()
		sr.lock.Unlock()
		if err != nil {
			http.Error(w, fmt.Sprintf("unknown serie '%s'", serie), http.StatusInternalServerError)
			return nil
		}
	}

	return sr
}

func (ms *MetricsServer) GetOffset(w http.ResponseWriter, r *http.Request) {
	sr := ms.getSerieReader("/get/offset/", w, r)
	if sr == nil {
		return
	}

	decoder := json.NewDecoder(r.Body)
	oreq := GetOffsetRequest{}
	err := decoder.Decode(&oreq)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not decode request '%s'", err), http.StatusBadRequest)
		return
	}

	if oreq.Entries <= 0 || oreq.Entries >= ms.MaxEntriesPerReply {
		oreq.Entries = ms.MaxEntriesPerReply
	}

	orep := GetOffsetReply{}
	orep.Request = oreq

	sr.lock.Lock()
	end := sr.reader.LastLocation()
	start := end.Minus(sr.reader, oreq.Entries)
	orep.Point, err = sr.reader.GetData(start, end, nil)
	sr.lock.Unlock()

	httpu.SendJsonReply(w, orep)
}

func (ms *MetricsServer) List(w http.ResponseWriter, r *http.Request) {
	keys := misc.StringKeysOrPanic(ms.sr)
	httpu.SendJsonReply(w, keys)
}
