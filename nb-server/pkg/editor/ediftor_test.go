package editor

import (
	"os"
	"testing"

	"github.com/nttcom/ksot/nb-server/pkg/model/orderedmap"
	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
	"github.com/stretchr/testify/assert"
)

var testEditor = NewEditorInterface()

var (
	testDeviceName = []string{"deviceA"}
	configPaths    = []string{
		"./testdata/deviceA.json",
	}
	diffResult = []map[string]*pathmap.DiffResult{
		{
			"deviceA": {
				Create: pathmap.PathMap{
					"/F[G=create_g]/H": pathmap.NewPathMapValueSafe([]string{"F[G=create_g]", "H"}, "create_h", make(map[string]string)),
				},
				Update: pathmap.PathMap{
					"/A/B/C":       pathmap.NewPathMapValueSafe([]string{"A", "B", "C"}, "update_c", make(map[string]string)),
					"/D/E[F=f1]/G": pathmap.NewPathMapValueSafe([]string{"D", "E[F=f1]", "G"}, "update_g", make(map[string]string)),
				},
				Delete: pathmap.PathMap{
					"/G[H=h]/H": pathmap.NewPathMapValueSafe([]string{"G[H=h]", "H"}, "h", make(map[string]string)),
				},
			},
		},
	}
	expectedConfigPaths = []string{
		"./testdata/deviceA_update.json",
	}
	expectedErrors = []error{
		nil,
	}
)

func TestEditConfigByPathmapDiff(t *testing.T) {
	t.Parallel()
	testConfigOrderedmaps := make([]map[string]orderedmap.OrderedmapInterfaces, 0)
	for i, v := range configPaths {
		testMap := make(map[string]orderedmap.OrderedmapInterfaces)
		jsonByte, err := os.ReadFile(v)
		assert.Nil(t, err)
		testOrderedmap, err := orderedmap.New(jsonByte)
		assert.Nil(t, err)
		testMap[testDeviceName[i]] = testOrderedmap
		testConfigOrderedmaps = append(testConfigOrderedmaps, testMap)
	}
	testExpectedOrderedmaps := make([]map[string]orderedmap.OrderedmapInterfaces, 0)
	for i, v := range expectedConfigPaths {
		testMap := make(map[string]orderedmap.OrderedmapInterfaces)
		jsonByte, err := os.ReadFile(v)
		assert.Nil(t, err)
		testOrderedmap, err := orderedmap.New(jsonByte)
		assert.Nil(t, err)
		testMap[testDeviceName[i]] = testOrderedmap
		testExpectedOrderedmaps = append(testExpectedOrderedmaps, testMap)
	}

	type test struct {
		arg1    map[string]orderedmap.OrderedmapInterfaces
		arg2    map[string]*pathmap.DiffResult
		want    map[string]orderedmap.OrderedmapInterfaces
		wantErr error
	}
	testNames := []string{
		"正常系",
	}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    testConfigOrderedmaps[i],
			arg2:    diffResult[i],
			want:    testExpectedOrderedmaps[i],
			wantErr: expectedErrors[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := testEditor.EditConfigByPathmapDiff(tt.arg1, tt.arg2)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, tt.arg1)
		})
	}
}
