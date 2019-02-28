package tsdb

import (
	//"time"
	"fmt"
	"strings"
	"sort"
	"strconv"
	"path/filepath"
	"os"
	//"os"
	//"syscall"
)

func ParseFileName(dbbasepath, fullname string) uint32 {
	if fullname == "" {
		return uint32(0)
	}
	stringid := strings.TrimPrefix(fullname, dbbasepath + "-")
	stringid = strings.TrimSuffix(stringid, ".data")
	stringid = strings.TrimSuffix(stringid, ".labels")
	id, err := strconv.ParseUint(stringid, 16, 32)
	if err != nil {
		return uint32(0)
	}
	return uint32(id)
}


func GetLastFile(dbbasepath string) string {
	pattern := dbbasepath + "-[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]" + ".data"
	matches, _ := filepath.Glob(pattern)
	sort.Strings(matches)
	if len(matches) > 0 {
		return matches[len(matches) - 1]
	}
	return ""
}

// Returns the id of the file to open for writes.
// It is either the id of the last file written to, or 1 in case no last file
// can be determined. 0 is a reserved value, which indicates errors / uninitialized.
func GetFileId(dbbasepath string) uint32 {
	id := ParseFileName(dbbasepath, GetLastFile(dbbasepath))
	if id == 0 {
		return 1
	}
	return id
}


func MakeFileName(dbbasepath string, number uint32, extension string) string {
	return fmt.Sprintf("%s-%08x.%s", dbbasepath, number, extension)
}

func MakeDataStoreFileName(dbbasepath string, number uint32) string {
	return MakeFileName(dbbasepath, number, "data")
}

func MakeLabelStoreFileName(dbbasepath string, number uint32) string {
	return MakeFileName(dbbasepath, number, "labels")
}

type Serie struct {
	Path string
	Id uint32

	DataWriterOptions
	LabelOptions

	dw *DataWriter
	ls *LabelStore
}

func NewSerie(dbbasepath string) *Serie {
	return &Serie{dbbasepath, 0, DefaultDataWriterOptions(), DefaultLabelOptions(), nil, nil}
}

func (serie *Serie) SetMode(mode os.FileMode) {
	serie.DataWriterOptions.Mode = mode
	serie.LabelOptions.Mode = mode
}

func (serie *Serie) Open() error {
	if serie.Id == 0 {
		serie.Id = GetFileId(serie.Path)
	}

	var err error
	for {
		serie.dw, err = OpenDataWriter(MakeDataStoreFileName(serie.Path, serie.Id), serie.DataWriterOptions)
		if err != nil {
			return err
		}

		if serie.dw.lpe == serie.LabelsPerEntry {
			break
		}

		serie.dw.Seal()
		serie.Id += 1
	}

	serie.ls, err = OpenLabels(MakeLabelStoreFileName(serie.Path, serie.Id), serie.LabelOptions)
	if err != nil {
		serie.dw.Close()
		return err
	}

	return nil
}

func (s *Serie) Append(time, value uint64, labels []string) error {
	for {
		if s.dw.Peek() {
			labelids := make([]LabelID, len(labels))
			for i, label := range labels {
				id, err := s.ls.CreateLabel(label)
				if err != nil {
					return err
				}
				labelids[i] = id
			}

			ok, _ := s.dw.Append(time, value, labelids)
			if ok {
				return nil
			}
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

func (s *Serie) Sync() {
	s.dw.Sync()
	s.ls.Sync()
}

func (s *Serie) Close() {
	s.dw.Close()
	s.ls.Close()
	s.Id = 0
}
