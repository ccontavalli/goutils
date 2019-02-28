package main

import (
	"flag"
	"github.com/ccontavalli/goutils/misc"
	"github.com/ccontavalli/goutils/tsdb"
	//"fmt"
	"log"
)

var (
	fl_serie = flag.String("serie", "", "Name of the serie to open. Generally, it is a "+
		"filesystem directory (/var/timeseries/) followed by the name of the serie "+
		"(/var/timeseries/load-over-time")
	fl_labelblock = flag.Int("labelblock", -1, "Size of a block to use for the labels "+
		"database. Defaults to 4Mb when <= 0")
	fl_labelsperentry = flag.Int("labelsperentry", -1, "Maximum number of labels per time entry "+
		"we will ever save. Defaults to 4 when < 0")
	fl_maxentries = flag.Int("maxentries", -1, "Maximum number of entries to store per file "+
		"before rotating it. Defaults to 604800 (a week of 1 second points) or ~20Mb")

	fl_action = flag.String("action", "add-value", "Action to perform. Can be: "+
		"add-value to add a single value (use --time, --value), list (to list values)")

	fl_time  = flag.Uint64("time", 0, "Time point to save in the database. Must be used with --value.")
	fl_value = flag.Uint64("value", 0, "Value to save in the database. Must be used with --time.")
	fl_label = misc.MultiString("label", nil, "Labels to associate to the point to save. Must be used with --value and --time.")
)

func AddValue() {
	if *fl_serie == "" {
		log.Fatalf("Must specify --serie, to indicate where to store the data")
	}

	s := tsdb.NewSerie(*fl_serie)
	if *fl_labelblock > 0 {
		s.LabelBlock = uint32(*fl_labelblock)
	}
	if *fl_labelsperentry >= 0 {
		if *fl_labelsperentry > 256 {
			log.Fatalf("Cannot store more than 256 labels per entry")
		}
		s.LabelsPerEntry = uint8(*fl_labelsperentry)
	}
	if *fl_maxentries > 0 {
		s.MaxEntries = uint32(*fl_maxentries)
	}

	if len(*fl_label) > int(s.LabelsPerEntry) {
		log.Fatalf("Too many labels requested via --lable, must be less than --labelsperentry")
	}
	if *fl_time == 0 || *fl_time == 0xffffffffffffffff {
		log.Fatalf("Time cannot be 0 or 0xfff... (-1) - those are reserved values")
	}

	err := s.Open()
	if err != nil {
		log.Fatalf("Failed to open time serie: %s", err)
	}

	err = s.Append(*fl_time, *fl_value, *fl_label)
	if err != nil {
		log.Fatalf("Failed to open time serie: %s", err)
	}
}

func List() {
}

func main() {
	flag.Parse()

	switch *fl_action {
	case "add-value":
		AddValue()
	case "list":
		List()
	default:
		log.Fatalf("Invalid action specified. Use --help to see list of valid actions")
	}

}
