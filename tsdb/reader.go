package tsdb

import (
	//"time"
	//"os"
	"fmt"
	"log"
	"os"
	"sort"
	//"syscall"
)

type shard struct {
	// The integer identifying this shard.
	fileid uint32
	// The first time stored in this shard.
	mintime uint64
	// The number of entries stored in this shard.
	entries int
	// The location of the shard in the shard index.
	index int

	dw *DataStore
	ls *LabelStore
}

type SerieReader struct {
	// Path of the serie, eg, /var/tsdb/kernel-memory.
	Path string
	// List of shards available. Note that writers can append
	// new shards any time, or old shards may be rotated out.
	shard []*shard
	// Shard indexes by name.
	byname map[string]*shard
}

func NewSerieReader(dbbasepath string) *SerieReader {
	return &SerieReader{dbbasepath, nil, make(map[string]*shard)}
}

func (shard *shard) Load(path string) error {
	if shard.dw != nil && shard.ls != nil {
		return nil
	}

	datafile := MakeDataStoreFileName(path, shard.fileid)
	dw, err := OpenDataStoreForReading(datafile)
	if err != nil {
		return err
	}

	labelfile := MakeLabelStoreFileName(path, shard.fileid)
	ls, err := OpenLabelsForReading(labelfile)
	if err != nil {
		dw.Close()
		shard.dw = nil
		return err
	}

	shard.dw = dw
	shard.ls = ls
	return nil
}

func (shard *shard) Unload() {
	shard.dw.Close()
	shard.ls.Close()

	shard.dw = nil
	shard.ls = nil
}

func (shard *shard) GetElements(s *SerieReader) int {
	lastshard := s.shard[len(s.shard)-1]
	if shard != lastshard {
		return shard.entries
	}
	lastshard.Load(s.Path)
	return lastshard.dw.GetEntries()
}

func (shard *shard) Next(s *SerieReader) *shard {
	if shard.index >= len(s.shard) {
		return nil
	}
	return s.shard[shard.index+1]
}

func (shard *shard) Prev(s *SerieReader) *shard {
	if shard.index <= 0 {
		return nil
	}
	return s.shard[shard.index-1]
}

func (shard *shard) IsLast(s *SerieReader) bool {
	return shard.index >= len(s.shard)
}

func (s *SerieReader) ReloadShards() error {
	// Check if the last shard filled up or was sealed. If it wasn't, there surely is no new shard to load.
	var lastshard *shard
	if len(s.shard) > 0 {
		lastshard = s.shard[len(s.shard)-1]
		lastshard.Load(s.Path)
		more, _ := lastshard.dw.PeekAppend()
		if more {
			return nil
		}
	}

	startid := uint32(1)
	if lastshard != nil {
		startid = lastshard.fileid
	}

	newshards := make([]*shard, len(s.shard))
	for fileid := startid; ; fileid++ {
		filename := MakeDataStoreFileName(s.Path, fileid)
		newshard, ok := s.byname[filename]
		if !ok {
			point, entries, err := PeekDataStore(MakeDataStoreFileName(s.Path, fileid))
			if err != nil {
				if os.IsNotExist(err) {
					break
				}
				return err
			}
			newshard = &shard{fileid, point.Time, entries, len(newshards), nil, nil}
			s.byname[filename] = newshard
		}
		newshards = append(newshards, newshard)
	}
	if len(newshards) <= 0 {
		return fmt.Errorf("serie not found - not a single shard in folder")
	}
	// TODO: we shoul garbage collect / unload old unused / infrequently used shards.
	s.shard = newshards
	return nil
}

func (s *SerieReader) Open() error {
	return s.ReloadShards()
}

type Point struct {
	Time, Value uint64
	Label       []string
}

type Location struct {
	shard   *shard
	element int
}

func (l *Location) Offset(s *SerieReader, value int) Location {
	if value == 0 {
		return *l
	}
	if value > 0 {
		return l.Plus(s, value)
	}
	return l.Minus(s, -value)
}

func (l *Location) Plus(s *SerieReader, value int) Location {
	lastshard := s.shard[len(s.shard)-1]
	for shard := l.shard; ; {
		elements := shard.GetElements(s)
		if elements > l.element+value {
			return Location{shard, l.element + value}
		}
		if shard == lastshard {
			return Location{lastshard, elements}
		}

		value -= elements
		shard = shard.Next(s)
	}
	// Should actually never be reached.
	return Location{lastshard, lastshard.GetElements(s)}
}

func (l *Location) Minus(s *SerieReader, value int) Location {
	if l.element > value {
		return Location{l.shard, l.element - value}
	}
	value -= l.element
	shard := l.shard.Prev(s)
	for shard != nil {
		if shard.entries >= value {
			return Location{shard, shard.entries - value}
		}
		value -= shard.entries
		shard = shard.Prev(s)
	}
	return Location{s.shard[0], 0}
}

type Summarizer func(points []Point, location Location, time, value uint64) []Point

func (s *SerieReader) GetLabels(location Location, labels []string) []string {
	shard := location.shard
	err := shard.Load(s.Path)
	if err != nil {
		return []string{}
	}

	labelids := shard.dw.GetLabels(shard.dw.GetOffset(location.element), nil)
	for _, labelid := range labelids {
		result, err := shard.ls.LoadString(LabelID(labelid))
		if err != nil {
			log.Printf("Corrupted database? LoadString lead to: %s", err)
		}
		labels = append(labels, result)
	}
	return labels
}

func (s *SerieReader) GetData(start, end Location, summarizer Summarizer) ([]Point, error) {
	if summarizer == nil {
		summarizer = func(points []Point, location Location, time, value uint64) []Point {
			return append(points, Point{time, value, s.GetLabels(location, nil)})
		}
	}

	maxelement := 0
	minelement := start.element
	points := []Point{}

	cursor := start.shard.index
	if s.shard[cursor] != start.shard {
		return []Point{}, fmt.Errorf("Start is now invalid - shard is gone")
	}
	last := end.shard.index
	if s.shard[last] != end.shard {
		return []Point{}, fmt.Errorf("End is now invalid - shard is gone")
	}
	if cursor > last {
		return []Point{}, fmt.Errorf("End < Start is invalid")
	}

	for ; cursor <= last; cursor++ {
		shard := s.shard[cursor]
		err := shard.Load(s.Path)
		if err != nil {
			continue
		}

		if cursor == last {
			maxelement = end.element
		} else {
			maxelement = shard.dw.GetEntries()
		}

		for j := minelement; j < maxelement; j++ {
			offset := shard.dw.GetOffset(j)
			time := shard.dw.GetTime(offset)
			value := shard.dw.GetTime(offset)

			points = summarizer(points, Location{shard, j}, time, value)
		}

		minelement = 0
	}
	return points, nil
}

type Finder func(time uint64) bool

// Returns the very first element in the time serie.
func (s *SerieReader) FirstLocation() Location {
	s.ReloadShards()

	return Location{s.shard[0], 0}
}

// Returns the last location in the time serie.
// This is one element past the last value stored, similar
// to what slice[len(slice)] in go would lead to.
// This is mostly useful to get the GetData arithmetic to work easily.
func (s *SerieReader) LastLocation() Location {
	s.ReloadShards()

	lastshard := s.shard[len(s.shard)-1]
	lastshard.Load(s.Path)

	return Location{lastshard, lastshard.dw.GetEntries()}
}

func (s *SerieReader) Find(finder Finder) Location {
	s.ReloadShards()

	minshard := sort.Search(len(s.shard), func(i int) bool {
		time := s.shard[i].mintime
		return finder(time)
	})

	shard := s.shard[minshard]
	err := shard.Load(s.Path)
	if err != nil {
		return Location{nil, 0}
	}

	element := sort.Search(shard.dw.GetEntries(), func(i int) bool {
		time := shard.dw.GetTime(shard.dw.GetOffset(i))
		return finder(time)
	})

	return Location{shard, element}
}
