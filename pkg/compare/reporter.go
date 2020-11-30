package compare

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

type DiffItem struct {
	Path string
	x reflect.Value
	y reflect.Value
}

func (diff *DiffItem) String() string {
	return fmt.Sprintf("%s (x: %s y: %s)", diff.Path, diff.x.Elem(), diff.y.Elem())
}

type Reporter struct {
	path        cmp.Path
	Differences []DiffItem
}

func (reporter *Reporter) PushStep(ps cmp.PathStep) {
	reporter.path = append(reporter.path, ps)
}

func (reporter *Reporter) PopStep() {
	reporter.path = reporter.path[:len(reporter.path)-1]
}

func (reporter *Reporter) Report(result cmp.Result) {
	if !result.Equal() {
		vx, vy := reporter.path.Last().Values()
		reporter.Differences = append(reporter.Differences, DiffItem{reporter.path.String(), vx, vy})
	}
}

func (reporter *Reporter) String() string {
	var diffs []string
	for _, diff := range reporter.Differences {
		diffs = append(diffs, diff.String())
	}
	return strings.Join(diffs, "\n")
}
