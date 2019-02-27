package tsdb

import (
	//"time"
	//"fmt"
	//"os"
	//"syscall"
)

type SerieOptions struct {
	DataOptions
	LabelOptions
}

func DefaultSerieOptions() SerieOptions {
	return SerieOptions{DefaultDataOptions(), DefaultLabelOptions()}
}

type Serie struct {
	*DataStore

	pl *LabelStore
	sl *LabelStore
}

func Open(dbbasepath string, options SerieOptions) (*Serie, error) {
	ds, err := OpenData(dbbasepath, options.DataOptions)
	if err != nil {
		return nil, err
	}

	pl, err := OpenLabels(dbbasepath + "-pl", options.LabelOptions)
	if err != nil {
		return nil, err
	}

	sl, err := OpenLabels(dbbasepath + "-sl", options.LabelOptions)
	if err != nil {
		return nil, err
	}

	return &Serie{ds, pl, sl}, nil
}

func (s *Serie) Append(time, value uint64, labels []string) {
	for _, l := range labels {
	}
}

func (s *Serie) Sync() {
	s.DataStore.Sync()
	s.pl.Sync()
	s.sl.Sync()
}

func (s *Serie) Close() {
	s.DataStore.Close()
	s.pl.Close()
	s.sl.Close()
}
