package orderedmap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configPaths = []string{
		"./testdata/deviceA.json",
	}
	testPaths = [][][]string{
		// update
		{
			// update
			{
				"A",
				"B",
				"C",
			},
			// create
			{
				"A",
				"B",
				"D",
			},
			// update list elm
			{
				"D",
				"E[F=f1]",
				"G",
			},
			{
				"L",
				"E[F=f1]",
				"M[N=n10]",
				"O",
			},
			// create list elm
			{
				"L",
				"E[F=create_f]",
				"G",
			},
			{
				"L",
				"E[F=f1]",
				"M[N=n10]",
				"P",
			},
			// create list value
			{
				"F[G=create_g]",
				"H",
			},
		},
	}
	testValues = [][]interface{}{
		{
			"update_c",
			"create_d",
			"update_g1",
			"update_o10",
			"create_g",
			"create_p10",
			"create_h",
		},
	}
	expectedConfigPaths = []string{
		"./testdata/deviceA_update.json",
	}
	expectedErrors = [][]error{
		{
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		},
	}
)

var (
	testDeleteConfigPaths = []string{
		"./testdata/deviceA.json",
	}
	testDeletePaths = [][][]string{
		{
			{
				"A",
				"B",
				"C",
			},
			// update list elm
			{
				"D",
				"E[F=f1]",
				"G",
			},
			{
				"L",
				"E[F=f1]",
				"M[N=n10]",
				"O",
			},
			{
				"G[H=h]",
				"H",
			},
		},
	}
	expectedTestDeleteConfigPaths = []string{
		"./testdata/deviceA_delete.json",
	}
	expectedTestDeleteErrors = [][]error{
		{
			nil,
			nil,
			nil,
			nil,
		},
	}
)

func TestRecursiveSet(t *testing.T) {
	t.Parallel()
	testConfigOrderedmaps := make([]Orderedmap, 0)
	for _, v := range configPaths {
		jsonByte, err := os.ReadFile(v)
		assert.Nil(t, err)
		testOrderedmap, err := New(jsonByte)
		assert.Nil(t, err)
		testConfigOrderedmaps = append(testConfigOrderedmaps, *testOrderedmap)
	}

	testExpectedOrderedmaps := make([]Orderedmap, 0)
	for _, v := range expectedConfigPaths {
		jsonByte, err := os.ReadFile(v)
		assert.Nil(t, err)
		testOrderedmap, err := New(jsonByte)
		assert.Nil(t, err)
		testExpectedOrderedmaps = append(testExpectedOrderedmaps, *testOrderedmap)
	}

	type test struct {
		arg1    Orderedmap
		arg2    [][]string
		arg3    []interface{}
		want    Orderedmap
		wantErr []error
	}
	testNames := []string{
		"正常系",
	}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    testConfigOrderedmaps[0],
			arg2:    testPaths[i],
			arg3:    testValues[i],
			want:    testExpectedOrderedmaps[i],
			wantErr: expectedErrors[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for i, v := range tt.arg2 {
				err := tt.arg1.RecursiveSet(v, tt.arg3[i])
				assert.Equal(t, tt.wantErr[i], err)
			}
			assert.Equal(t, *tt.want.Value, *tt.arg1.Value)
		})
	}
}

func TestRecursiveDelete(t *testing.T) {
	t.Parallel()
	testConfigOrderedmaps := make([]Orderedmap, 0)
	for _, v := range testDeleteConfigPaths {
		jsonByte, err := os.ReadFile(v)
		assert.Nil(t, err)
		testOrderedmap, err := New(jsonByte)
		assert.Nil(t, err)
		testConfigOrderedmaps = append(testConfigOrderedmaps, *testOrderedmap)
	}

	testExpectedOrderedmaps := make([]Orderedmap, 0)
	for _, v := range expectedTestDeleteConfigPaths {
		jsonByte, err := os.ReadFile(v)
		assert.Nil(t, err)
		testOrderedmap, err := New(jsonByte)
		assert.Nil(t, err)
		testExpectedOrderedmaps = append(testExpectedOrderedmaps, *testOrderedmap)
	}

	type test struct {
		arg1    Orderedmap
		arg2    [][]string
		want    Orderedmap
		wantErr []error
	}
	testNames := []string{
		"正常系",
	}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:    testConfigOrderedmaps[0],
			arg2:    testDeletePaths[i],
			want:    testExpectedOrderedmaps[i],
			wantErr: expectedTestDeleteErrors[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for i, v := range tt.arg2 {
				err := tt.arg1.RecursiveDelete(v)
				assert.Equal(t, tt.wantErr[i], err)
			}
			assert.Equal(t, *tt.want.Value, *tt.arg1.Value)
		})
	}
}
