package composite

import (
	"testing"

	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
	"github.com/stretchr/testify/assert"
)

var testComposite = NewCompositeInterface()

var (
	serviceDeviceToPathmaps = []map[string]map[string]pathmap.PathMapInterface{
		{
			"serviceA": {
				"deviceA": pathmap.PathMap{
					"/string": pathmap.NewPathMapValueSafe([]string{"string"}, "update_a", make(map[string]string)),
					"/bool":   pathmap.NewPathMapValueSafe([]string{"bool"}, true, make(map[string]string)),
					"/num":    pathmap.NewPathMapValueSafe([]string{"num"}, 100, make(map[string]string)),
				},
			},
		},
	}
	oldServiceToPathmaps = []map[string]map[string]pathmap.PathMapInterface{
		{
			"deviceA": {
				"serviceA": pathmap.PathMap{
					"/string": pathmap.NewPathMapValueSafe([]string{"string"}, "a", make(map[string]string)),
					"/bool":   pathmap.NewPathMapValueSafe([]string{"bool"}, true, make(map[string]string)),
				},
				"serviceB": pathmap.PathMap{
					"/string_list_B": pathmap.NewPathMapValueSafe([]string{"string_list_B"}, []string{"a", "b", "c"}, make(map[string]string)),
				},
			},
		},
	}
	wantDeviceToPathmap = []map[string]pathmap.PathMapInterface{
		{
			"deviceA": pathmap.PathMap{
				"/string":        pathmap.NewPathMapValueSafe([]string{"string"}, "update_a", make(map[string]string)),
				"/bool":          pathmap.NewPathMapValueSafe([]string{"bool"}, true, make(map[string]string)),
				"/num":           pathmap.NewPathMapValueSafe([]string{"num"}, 100, make(map[string]string)),
				"/string_list_B": pathmap.NewPathMapValueSafe([]string{"string_list_B"}, []string{"a", "b", "c"}, make(map[string]string)),
			},
		},
	}
	wantDeviceToOldPathmap = []map[string]pathmap.PathMapInterface{
		{
			"deviceA": pathmap.PathMap{
				"/string":        pathmap.NewPathMapValueSafe([]string{"string"}, "a", make(map[string]string)),
				"/bool":          pathmap.NewPathMapValueSafe([]string{"bool"}, true, make(map[string]string)),
				"/string_list_B": pathmap.NewPathMapValueSafe([]string{"string_list_B"}, []string{"a", "b", "c"}, make(map[string]string)),
			},
		},
	}
	compositePathMapWantsErrs = []error{
		nil,
	}
)

func TestCompositPathmaps(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1    map[string]map[string]pathmap.PathMapInterface
		want    map[string]pathmap.PathMapInterface
		wantErr error
	}
	testNames := []string{"正常系: composite成功時"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    oldServiceToPathmaps[i],
			want:    wantDeviceToOldPathmap[i],
			wantErr: compositePathMapWantsErrs[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := testComposite.CompositePathmaps(tt.arg1)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCompositeAndUpdateAndReplaceKeyPathmaps(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1    map[string]map[string]pathmap.PathMapInterface
		arg2    map[string]map[string]pathmap.PathMapInterface
		want    map[string]pathmap.PathMapInterface
		wantErr error
	}
	testNames := []string{"正常系: composite成功時"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    serviceDeviceToPathmaps[i],
			arg2:    oldServiceToPathmaps[i],
			want:    wantDeviceToPathmap[i],
			wantErr: compositePathMapWantsErrs[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := testComposite.CompositeAndUpdateAndReplaceKeyPathmaps(tt.arg1, tt.arg2)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, result)
		})
	}
}
