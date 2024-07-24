package pathmap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testPathmaps = []PathMap{
		{
			"/string": &PathMapValue{
				value:  "a",
				path:   []string{"string"},
				option: make(map[string]string),
			},
			"/bool": &PathMapValue{
				value:  true,
				path:   []string{"bool"},
				option: make(map[string]string),
			},
			"/num": &PathMapValue{
				value:  100,
				path:   []string{"num"},
				option: make(map[string]string),
			},
			"/string_list": &PathMapValue{
				value:  []string{"a", "b", "c"},
				path:   []string{"string_list"},
				option: make(map[string]string),
			},
		},
		{
			"/string": &PathMapValue{
				value:  "a",
				path:   []string{"string"},
				option: make(map[string]string),
			},
			"/bool": &PathMapValue{
				value:  true,
				path:   []string{"bool"},
				option: make(map[string]string),
			},
			"/num": &PathMapValue{
				value:  100,
				path:   []string{"num"},
				option: make(map[string]string),
			},
			"/string_list": &PathMapValue{
				value:  []string{"a", "b", "c"},
				path:   []string{"string_list"},
				option: make(map[string]string),
			},
		},
	}
	testCompositePathmapLists = [][]PathMapInterface{
		{
			PathMap{
				"/string": &PathMapValue{
					value:  "a",
					path:   []string{"string"},
					option: make(map[string]string),
				},
				"/bool": &PathMapValue{
					value:  true,
					path:   []string{"bool"},
					option: make(map[string]string),
				},
				"/num": &PathMapValue{
					value:  100,
					path:   []string{"num"},
					option: make(map[string]string),
				},
				"/string_list": &PathMapValue{
					value:  []string{"a", "b", "c", "d", "e", "a", "b"},
					path:   []string{"string_list"},
					option: make(map[string]string),
				},
				"/num_list": &PathMapValue{
					value:  []float64{1, 2, 3, 4, 5},
					path:   []string{"num_list"},
					option: make(map[string]string),
				},
			},
		},
		{
			PathMap{
				"/string": &PathMapValue{
					value:  "a",
					path:   []string{"string"},
					option: make(map[string]string),
				},
				"/bool": &PathMapValue{
					value:  false,
					path:   []string{"bool"},
					option: make(map[string]string),
				},
				"/num": &PathMapValue{
					value:  100,
					path:   []string{"num"},
					option: make(map[string]string),
				},
				"/string_list": &PathMapValue{
					value:  []string{"a", "b", "c"},
					path:   []string{"string_list"},
					option: make(map[string]string),
				},
			},
		},
	}
	compositeWantPathmaps = []PathMapInterface{
		PathMap{
			"/string": &PathMapValue{
				value:  "a",
				path:   []string{"string"},
				option: make(map[string]string),
			},
			"/bool": &PathMapValue{
				value:  true,
				path:   []string{"bool"},
				option: make(map[string]string),
			},
			"/num": &PathMapValue{
				value:  100,
				path:   []string{"num"},
				option: make(map[string]string),
			},
			"/string_list": &PathMapValue{
				value:  []string{"a", "b", "c", "d", "e"},
				path:   []string{"string_list"},
				option: make(map[string]string),
			},
			"/num_list": &PathMapValue{
				value:  []float64{1, 2, 3, 4, 5},
				path:   []string{"num_list"},
				option: make(map[string]string),
			},
		},
		PathMap{
			"/string": &PathMapValue{
				value:  "a",
				path:   []string{"string"},
				option: make(map[string]string),
			},
			"/bool": &PathMapValue{
				value:  true,
				path:   []string{"bool"},
				option: make(map[string]string),
			},
			"/num": &PathMapValue{
				value:  100,
				path:   []string{"num"},
				option: make(map[string]string),
			},
			"/string_list": &PathMapValue{
				value:  []string{"a", "b", "c"},
				path:   []string{"string_list"},
				option: make(map[string]string),
			},
		},
	}
	compositePathMapWantsErrs = []error{
		nil,
		fmt.Errorf("CompositePathMap: %w", fmt.Errorf("mergePathMapValue: %w", fmt.Errorf("checkConflictOrMergeListForPathMapValue: conflict error %v : %v", true, false))),
	}
)

var (
	oldPathmaps = []PathMap{
		{
			"/A/B/C": &PathMapValue{
				value:  0,
				path:   []string{"A", "B", "C"},
				option: make(map[string]string),
			},
			"/D/E[name=hoge]/F": &PathMapValue{
				value:  "a",
				path:   []string{"D", "E[name=hoge]", "F"},
				option: make(map[string]string),
			},
			"/G/H/I": &PathMapValue{
				value:  []int{1, 2, 3},
				path:   []string{"G", "H", "I"},
				option: make(map[string]string),
			},
			"/J/K/L": &PathMapValue{
				value:  []string{"a", "b", "c"},
				path:   []string{"J", "K", "L"},
				option: make(map[string]string),
			},
		},
	}
	newPathmaps = []PathMap{
		{
			"/A/B/C": &PathMapValue{
				value:  0,
				path:   []string{"A", "B", "C"},
				option: make(map[string]string),
			},
			// delete
			// "/D/E[name=hoge]/F"
			// update
			"/G/H/I": &PathMapValue{
				value:  []int{1, 2, 3, 4, 5},
				path:   []string{"G", "H", "I"},
				option: make(map[string]string),
			},
			// update
			"/J/K/L": &PathMapValue{
				value:  []string{"a"},
				path:   []string{"J", "K", "L"},
				option: make(map[string]string),
			},
			// create
			"/M/N/O": &PathMapValue{
				value:  []float64{1.0, 2.0, 3.0},
				path:   []string{"M", "N", "O"},
				option: make(map[string]string),
			},
		},
	}
	expectedDiffPathMaps = []DiffResult{
		{
			Create: PathMap{
				"/M/N/O": &PathMapValue{
					value:  []float64{1.0, 2.0, 3.0},
					path:   []string{"M", "N", "O"},
					option: make(map[string]string),
				},
			},
			Update: PathMap{
				"/G/H/I": &PathMapValue{
					value:  []int{1, 2, 3, 4, 5},
					path:   []string{"G", "H", "I"},
					option: make(map[string]string),
				},
				"/J/K/L": &PathMapValue{
					value:  []string{"a"},
					path:   []string{"J", "K", "L"},
					option: make(map[string]string),
				},
			},
			Delete: PathMap{
				"/D/E[name=hoge]/F": &PathMapValue{
					value:  "a",
					path:   []string{"D", "E[name=hoge]", "F"},
					option: make(map[string]string),
				},
			},
		},
	}
	wantErrs = []error{
		nil,
	}
)

var (
	testNewMaps = []map[string]any{
		{
			"/openconfig-platform:components/component[name=oc1]/openconfig-platform-transceiver:transceiver/config/enabled": "false",
			"/openconfig-platform:components/component[name=oc2]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
			"/openconfig-platform:components/component[name=oc3]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
			"/openconfig-platform:components/component[name=oc4]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
			"/openconfig-platform:components/component[name=oc5]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
		},
		{
			"/openconfig-platform:components/component[name=oc1]/openconfig-platform-transceiver:transceiver/config/enabled": "false",
			"/openconfig-platform:components/component[name=oc2]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
			"/openconfig-platform:components/component[name=oc3]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
			"/openconfig-platform:components/component[name=oc4]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
			"/openconfig-platform:components/component[name=oc5]/openconfig-platform-transceiver:transceiver/config/enabled": "true",
		},
	}
	wantNewErrs = []error{
		nil,
	}
)

func TestDiffPathMap(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1    PathMap
		arg2    PathMap
		want    DiffResult
		wantErr error
	}
	testNames := []string{"正常系: 文字列、bool、intを要素として含むpathmapのdiff取得"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    oldPathmaps[i],
			arg2:    newPathmaps[i],
			want:    expectedDiffPathMaps[i],
			wantErr: wantErrs[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			diffResult, err := tt.arg1.Diff(tt.arg2)
			assert.Nil(t, err)
			assert.Equal(t, tt.want, *diffResult)
		})
	}
}

func TestCompositePathMap(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1    PathMap
		arg2    []PathMapInterface
		want    PathMapInterface
		wantErr error
	}
	testNames := []string{"正常系: composite成功時", "異常系: conflect時"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    testPathmaps[i],
			arg2:    testCompositePathmapLists[i],
			want:    compositeWantPathmaps[i],
			wantErr: compositePathMapWantsErrs[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := tt.arg1.Composite(tt.arg2)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, tt.arg1)
		})
	}
}

func TestNewathMap(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1    map[string]any
		wantErr error
	}
	testNames := []string{"正常系: composite成功時"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    testNewMaps[i],
			wantErr: wantNewErrs[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			x, err := NewPathMap(tt.arg1)
			fmt.Println(x)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
