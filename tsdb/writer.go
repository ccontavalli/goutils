package tsdb

import (
	//"time"
	"os"
	//"os"
	//"syscall"
)

type SerieWriter struct {
	Path string
	Id   uint32

	DataStoreOptions
	LabelOptions

	dw *DataStore
	ls *LabelStore
}

func NewSerieWriter(dbbasepath string) *SerieWriter {
	return &SerieWriter{dbbasepath, 0, DefaultDataStoreOptions(), DefaultLabelOptions(), nil, nil}
}

func (serie *SerieWriter) SetMode(mode os.FileMode) {
	serie.DataStoreOptions.Mode = mode
	serie.LabelOptions.Mode = mode
}

func (serie *SerieWriter) Open() error {
	if serie.Id == 0 {
		serie.Id = GetFileId(serie.Path)
	}

	var err error
	for {
		serie.dw, err = OpenDataStoreForWriting(MakeDataStoreFileName(serie.Path, serie.Id), serie.DataStoreOptions)
		if err != nil {
			return err
		}

		if serie.dw.lpe == serie.LabelsPerEntry {
			break
		}

		serie.dw.Seal()
		serie.Id += 1
	}

	serie.ls, err = OpenLabelsForWriting(MakeLabelStoreFileName(serie.Path, serie.Id), serie.LabelOptions)
	if err != nil {
		serie.dw.Close()
		return err
	}

	return nil
}

func (s *SerieWriter) Append(time, value uint64, labels []string) error {
	for {
		labelids := []LabelID{}
		// This tries to avoid creating labels associated to this store if the store is full.
		if len(labels) > 0 {
			more, _ := s.dw.PeekAppend()
			if more {
				labelids = make([]LabelID, len(labels))
				for i, label := range labels {
					id, err := s.ls.CreateLabel(label)
					if err != nil {
						return err
					}
					labelids[i] = id
				}
			}
		}

		ok, _ := s.dw.Append(time, value, labelids)
		if ok {
			return nil
		}

		s.dw.Seal()
		s.ls.Seal()

		s.Id += 1

		err := s.Open()
		if err != nil {
			return err
		}
	}
}

func (s *SerieWriter) Sync() {
	s.dw.Sync()
	s.ls.Sync()
}

func (s *SerieWriter) Close() {
	s.dw.Close()
	s.ls.Close()
	s.Id = 0
}
