package diff

import (
	"testing"

	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
	"github.com/stretchr/testify/assert"
)

var testDiff = NewDiffInterface()

var (
	oldDeviceToPathmaps = []map[string]pathmap.PathMapInterface{
		{
			"deviceA": pathmap.PathMap{
				"/string": pathmap.NewPathMapValueSafe([]string{"string"}, "a", make(map[string]string)),
				"/bool":   pathmap.NewPathMapValueSafe([]string{"bool"}, true, make(map[string]string)),
				"/num":    pathmap.NewPathMapValueSafe([]string{"num"}, 100, make(map[string]string)),
			},
		},
	}
	newDeviceToPathmaps = []map[string]pathmap.PathMapInterface{
		{
			"deviceA": pathmap.PathMap{
				"/string":     pathmap.NewPathMapValueSafe([]string{"string"}, "update_a", make(map[string]string)),
				"/num":        pathmap.NewPathMapValueSafe([]string{"num"}, 100, make(map[string]string)),
				"/create_num": pathmap.NewPathMapValueSafe([]string{"create_num"}, 10000, make(map[string]string)),
			},
		},
	}
	wantDiffResult = []map[string]*pathmap.DiffResult{
		{
			"deviceA": {
				Create: pathmap.PathMap{
					"/create_num": pathmap.NewPathMapValueSafe([]string{"create_num"}, 10000, make(map[string]string)),
				},
				Update: pathmap.PathMap{
					"/string": pathmap.NewPathMapValueSafe([]string{"string"}, "update_a", make(map[string]string)),
				},
				Delete: pathmap.PathMap{
					"/bool": pathmap.NewPathMapValueSafe([]string{"bool"}, true, make(map[string]string)),
				},
			},
		},
	}
	wantErrs = []error{
		nil,
	}
)

func TestDiffPathMap(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1    map[string]pathmap.PathMapInterface
		arg2    map[string]pathmap.PathMapInterface
		want    map[string]*pathmap.DiffResult
		wantErr error
	}
	testNames := []string{"正常系: 文字列、bool、intを要素として含むpathmapのdiff取得"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    oldDeviceToPathmaps[i],
			arg2:    newDeviceToPathmaps[i],
			want:    wantDiffResult[i],
			wantErr: wantErrs[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := testDiff.DiffPathmaps(tt.arg1, tt.arg2)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, result)
		})
	}
}
