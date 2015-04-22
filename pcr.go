package ts

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PcrInfo struct {
	Pos int64
	Pcr int64
}

func CheckPcrInterval(root string, pcrPid int, pcrList []PcrInfo) {
	fname := filepath.Join(root, "pcr-"+strconv.Itoa(pcrPid)+".csv")
	w, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	var prevPcr int64 = -1
	var diff int64
	// millisecond
	var maxInterval int64 = 40 * 27000
	for _, pcrInfo := range pcrList {
		pcrPos, pcr := pcrInfo.Pos, pcrInfo.Pcr
		if prevPcr != -1 {
			diff = pcr - prevPcr
		}
		prevPcr = pcr

		cols := []string{
			strconv.FormatInt(pcrPos, 10),
			strconv.FormatInt(pcr, 10),
			strconv.FormatInt(diff, 10),
		}
		if diff > maxInterval {
			cols = append(cols, "ErrInterval")
		}
		fmt.Fprintln(w, strings.Join(cols, ", "))
	}
}
