package tsdb

import (
	//"time"
	//"os"
	"log"
	"sort"
	//"os"
	//"syscall"
)

type shard struct {
	mintime uint64

	dw *DataWriter
	ls *LabelStore
}

type SerieReader struct {
	Path string

	DataWriterOptions
	LabelOptions

	shard []shard
}

func NewSerieReader(dbbasepath string) *SerieReader {
	return &SerieReader{dbbasepath, DefaultDataWriterOptions(), DefaultLabelOptions(), nil}
}

func (s *SerieReader) ReloadShards() error {
	files := GetDataFiles(s.Path)
	for _, filename := range files {
		fileid := ParseFileName(s.Path, filename)
		if fileid == 0 {
			continue
		}

	}
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
	shard   int
	element int
}

type Summarizer func(points []Point, location Location, time, value uint64) []Point

func (s *SerieReader) GetLabels(location Location, labels []string) []string {
	shard := s.shard[location.shard]
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
	s.ReloadShards()

	if summarizer == nil {
		summarizer = func(points []Point, location Location, time, value uint64) []Point {
			return append(points, Point{time, value, s.GetLabels(location, nil)})
		}
	}

	maxelement := 0
	minelement := start.element
	points := []Point{}
	for i := start.shard; i <= end.shard; i++ {
		shard := s.shard[i]
		if i == end.shard {
			maxelement = end.element
		} else {
			maxelement = shard.dw.GetEntries()
		}

		for j := minelement; j <= maxelement; j++ {
			offset := shard.dw.GetOffset(j)
			time := shard.dw.GetTime(offset)
			value := shard.dw.GetTime(offset)

			points = summarizer(points, Location{i, j}, time, value)
		}

		minelement = 0
	}
	return points, nil
}

type Finder func(time uint64) bool

func (s *SerieReader) GetLocation(finder Finder) Location {
	s.ReloadShards()

	minshard := sort.Search(len(s.shard), func(i int) bool {
		time := s.shard[i].mintime
		return finder(time)
	})

	shard := s.shard[minshard]
	element := sort.Search(shard.dw.GetEntries(), func(i int) bool {
		time := shard.dw.GetTime(shard.dw.GetOffset(i))
		return finder(time)
	})

	return Location{minshard, element}
}

// Common use cases:
// - show the last hour of points
// - show the last n points
// - show the hour from to
