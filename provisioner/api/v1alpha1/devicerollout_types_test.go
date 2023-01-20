/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package v1alpha1_test

import (
	"testing"

	apiv1alpha1 "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestDeviceConfigMap_Equal(t *testing.T) {
	m := apiv1alpha1.DeviceConfigMap{
		"d1": apiv1alpha1.DeviceConfig{
			Checksum:    "Checksum1",
			GitRevision: "rev1",
		},
		"d2": apiv1alpha1.DeviceConfig{
			Checksum:    "Checksum2",
			GitRevision: "rev2",
		},
	}
	testcases := []struct {
		name  string
		given apiv1alpha1.DeviceConfigMap
		want  bool
	}{
		{
			name: "the same one without copy",
			given: apiv1alpha1.DeviceConfigMap{
				"d1": apiv1alpha1.DeviceConfig{
					Checksum:    "Checksum1",
					GitRevision: "rev1",
				},
				"d2": apiv1alpha1.DeviceConfig{
					Checksum:    "Checksum2",
					GitRevision: "rev2",
				},
			},
			want: true,
		},
		{
			name: "the different one without copy",
			given: apiv1alpha1.DeviceConfigMap{
				"d1": apiv1alpha1.DeviceConfig{
					Checksum:    "Checksum1",
					GitRevision: "rev2",
				},
				"d2": apiv1alpha1.DeviceConfig{
					Checksum:    "Checksum2",
					GitRevision: "rev2",
				},
			},
			want: false,
		},
		{
			name:  "the same one with deepcopy",
			given: m.DeepCopy(),
			want:  true,
		},
	}
	for _, tc := range testcases {
		assert.Equal(t, tc.want, m.Equal(tc.given))
	}
}

func TestDeviceRolloutStatus_IsRunning(t *testing.T) {
	s := apiv1alpha1.DeviceRolloutStatus{}
	assert.False(t, s.IsRunning())
	s.Status = apiv1alpha1.RolloutStatusRunning
	assert.True(t, s.IsRunning())
}

func TestDeviceRolloutStatus_DeviceStatus(t *testing.T) {
	tests := []struct {
		name            string
		given           apiv1alpha1.DeviceRolloutStatus
		wantTxCompleted bool
		wantTxFailed    bool
		wantTxRunning   bool
		wantTxIdle      bool
	}{
		{
			"false: not initialized",
			apiv1alpha1.DeviceRolloutStatus{},
			false,
			false,
			false,
			false,
		},
		{
			"all completed",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"completed":  apiv1alpha1.DeviceStatusCompleted,
					"completed2": apiv1alpha1.DeviceStatusCompleted,
					"purged":     apiv1alpha1.DeviceStatusPurged,
				},
			},
			true,
			false,
			false,
			true,
		},
		{
			"all running or completed",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"completed": apiv1alpha1.DeviceStatusCompleted,
					"running":   apiv1alpha1.DeviceStatusRunning,
					"purged":    apiv1alpha1.DeviceStatusPurged,
				},
			},
			false,
			false,
			true,
			false,
		},
		{
			"some failed",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"failed":    apiv1alpha1.DeviceStatusFailed,
					"running":   apiv1alpha1.DeviceStatusRunning,
					"completed": apiv1alpha1.DeviceStatusCompleted,
					"purged":    apiv1alpha1.DeviceStatusPurged,
				},
			},
			false,
			true,
			true,
			false,
		},
		{
			"some connection error",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"connError": apiv1alpha1.DeviceStatusConnectionError,
					"running":   apiv1alpha1.DeviceStatusRunning,
					"completed": apiv1alpha1.DeviceStatusCompleted,
					"purged":    apiv1alpha1.DeviceStatusPurged,
				},
			},
			false,
			true,
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantTxCompleted, tt.given.IsTxCompleted())
			assert.Equal(t, tt.wantTxFailed, tt.given.IsTxFailed())
			assert.Equal(t, tt.wantTxRunning, tt.given.IsTxRunning())
			assert.Equal(t, tt.wantTxIdle, tt.given.IsTxIdle())
		})
	}
}

func TestDeviceRolloutStatus_StartTx(t *testing.T) {
	tests := []struct {
		name  string
		given apiv1alpha1.DeviceRolloutStatus
		want  map[string]apiv1alpha1.DeviceStatus
	}{
		{
			"init statusMap without record",
			apiv1alpha1.DeviceRolloutStatus{
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{},
				DeviceStatusMap:        nil,
			},
			map[string]apiv1alpha1.DeviceStatus{},
		},
		{
			"init statusMap ",
			apiv1alpha1.DeviceRolloutStatus{
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"new1": apiv1alpha1.DeviceConfig{},
					"new2": apiv1alpha1.DeviceConfig{},
				},
				DeviceStatusMap: nil,
			},
			map[string]apiv1alpha1.DeviceStatus{
				"new1": apiv1alpha1.DeviceStatusRunning,
				"new2": apiv1alpha1.DeviceStatusRunning,
			},
		},
		{
			"update statusMap along with purging",
			apiv1alpha1.DeviceRolloutStatus{
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"curr1": apiv1alpha1.DeviceConfig{},
					"curr2": apiv1alpha1.DeviceConfig{},
					"new":   apiv1alpha1.DeviceConfig{},
				},
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"gone":  apiv1alpha1.DeviceStatusCompleted,
					"curr1": apiv1alpha1.DeviceStatusCompleted,
					"curr2": apiv1alpha1.DeviceStatusFailed,
				},
			},
			map[string]apiv1alpha1.DeviceStatus{
				"gone":  apiv1alpha1.DeviceStatusPurged,
				"curr1": apiv1alpha1.DeviceStatusRunning,
				"curr2": apiv1alpha1.DeviceStatusRunning,
				"new":   apiv1alpha1.DeviceStatusRunning,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.given.StartTx()
			assert.Equal(t, tt.want, tt.given.DeviceStatusMap)
		})
	}
}

func TestDeviceRolloutStatus_ResolveNextDeviceConfig(t *testing.T) {
	desired := apiv1alpha1.DeviceConfig{GitRevision: "desired"}
	prev := apiv1alpha1.DeviceConfig{GitRevision: "prev"}

	tests := []struct {
		name   string
		device string
		phase  apiv1alpha1.RolloutPhase
		want   *apiv1alpha1.DeviceConfig
	}{
		{
			"ok: healthy",
			"device1",
			apiv1alpha1.RolloutPhaseHealthy,
			&desired,
		},
		{
			"ok: rollback",
			"device1",
			apiv1alpha1.RolloutPhaseRollback,
			&prev,
		},
		{
			"err: not set",
			"device1",
			"",
			nil,
		},
		{
			"err: healthy but not existing device",
			"not-exist",
			apiv1alpha1.RolloutPhaseHealthy,
			nil,
		},
		{
			"err: rollback but not existing device",
			"not-exist",
			apiv1alpha1.RolloutPhaseRollback,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := apiv1alpha1.DeviceRolloutStatus{
				Phase: tt.phase,
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"device1": desired,
				},
				PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"device1": prev,
				},
			}

			got := s.ResolveNextDeviceConfig(tt.device)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeviceRolloutStatus_GetDeviceStatus(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		s := apiv1alpha1.DeviceRolloutStatus{}
		assert.Equal(t, apiv1alpha1.DeviceStatusUnknown, s.GetDeviceStatus("not-exist"))
	})

	t.Run("record not set", func(t *testing.T) {
		s := apiv1alpha1.DeviceRolloutStatus{
			DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{},
		}
		s.SetDeviceStatus("test", apiv1alpha1.DeviceStatusRunning)
		assert.Equal(t, apiv1alpha1.DeviceStatusUnknown, s.GetDeviceStatus("not-exist"))
	})

	t.Run("record set", func(t *testing.T) {
		s := apiv1alpha1.DeviceRolloutStatus{
			DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{},
		}
		s.SetDeviceStatus("test", apiv1alpha1.DeviceStatusRunning)
		assert.Equal(t, apiv1alpha1.DeviceStatusRunning, s.GetDeviceStatus("test"))
	})
}

func TestDeviceRollout_UpdateStatus(t *testing.T) {
	desired := apiv1alpha1.DeviceConfig{GitRevision: "desired"}
	curr := apiv1alpha1.DeviceConfig{GitRevision: "curr"}
	prev := apiv1alpha1.DeviceConfig{GitRevision: "prev"}

	t.Run("running", func(t *testing.T) {
		tests := []struct {
			name        string
			given       apiv1alpha1.DeviceRollout
			want        apiv1alpha1.DeviceRolloutStatus
			wantChanged bool
		}{
			{
				"on completed",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"completed": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"completed": apiv1alpha1.DeviceStatusCompleted,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusCompleted,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"completed": apiv1alpha1.DeviceStatusCompleted,
					},
				},
				true,
			},
			{
				"on running",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"running": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"running": apiv1alpha1.DeviceStatusRunning,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusRunning,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"running": apiv1alpha1.DeviceStatusRunning,
					},
				},
				false,
			},
			{
				"on failed healthy",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"failed": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"failed": apiv1alpha1.DeviceStatusFailed,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseRollback,
					Status: apiv1alpha1.RolloutStatusRunning,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"failed": apiv1alpha1.DeviceStatusRunning,
					},
				},
				true,
			},
			{
				"on failed rollback",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseRollback,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"failed": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"failed": apiv1alpha1.DeviceStatusFailed,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseRollback,
					Status: apiv1alpha1.RolloutStatusFailed,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"failed": apiv1alpha1.DeviceStatusFailed,
					},
				},
				true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := tt.given.DeepCopy()
				got := s.UpdateStatus()
				assert.Equal(t, tt.wantChanged, got)
				assert.Equal(t, tt.want.Phase, s.Status.Phase)
				assert.Equal(t, tt.want.Status, s.Status.Status)
				assert.Equal(t, tt.want.DeviceStatusMap, s.Status.DeviceStatusMap)
				// check not changed
				assert.Equal(t, tt.given.Status.DesiredDeviceConfigMap, s.Status.DesiredDeviceConfigMap)
			})
		}
	})

	t.Run("idle", func(t *testing.T) {
		t.Run("spec not updated", func(t *testing.T) {
			oldDr := apiv1alpha1.DeviceRollout{
				Spec: apiv1alpha1.DeviceRolloutSpec{
					DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": curr,
					},
				},
				Status: apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusCompleted,
					DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": curr,
					},
					PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": prev,
					},
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"device1": apiv1alpha1.DeviceStatusCompleted,
					},
				},
			}
			newDr := oldDr.DeepCopy()
			got := newDr.UpdateStatus()
			assert.False(t, got)
			assert.Equal(t, oldDr.Status, newDr.Status)
		})

		tests := []struct {
			name        string
			given       apiv1alpha1.DeviceRollout
			want        apiv1alpha1.DeviceRolloutStatus
			wantChanged bool
		}{
			{
				"spec updated: on healthy",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusCompleted,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": curr,
						},
						PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": prev,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"device1": apiv1alpha1.DeviceStatusCompleted,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusRunning,
					DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": desired,
					},
					PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": curr,
					},
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"device1": apiv1alpha1.DeviceStatusRunning,
					},
				},
				true,
			},
			{
				"spec updated: on rollback",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseRollback,
						Status: apiv1alpha1.RolloutStatusFailed,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": curr,
						},
						PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": prev,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"device1": apiv1alpha1.DeviceStatusFailed,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusRunning,
					DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": desired,
					},
					PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": prev,
					},
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"device1": apiv1alpha1.DeviceStatusRunning,
					},
				},
				true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := tt.given.UpdateStatus()
				assert.Equal(t, tt.wantChanged, got)
				assert.Equal(t, tt.want, tt.given.Status)
			})
		}
	})
}
