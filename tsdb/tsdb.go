package tsdb

import (
	//"time"
	//"fmt"
	//"os"
	//"syscall"
)

type SerieOptions struct {
	DataWriterOptions
	LabelOptions
}

func DefaultSerieOptions() SerieOptions {
	return SerieOptions{DefaultDataWriterOptions(), DefaultLabelOptions()}
}

type Serie struct {
	*DataWriter

	pl *LabelStore
	sl *LabelStore
}

func Open(dbbasepath string, options SerieOptions) (*Serie, error) {
	ds, err := OpenDataWriter(dbbasepath, options.DataWriterOptions)
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
	s.DataWriter.Sync()
	s.pl.Sync()
	s.sl.Sync()
}

func (s *Serie) Close() {
	s.DataWriter.Close()
	s.pl.Close()
	s.sl.Close()
}
