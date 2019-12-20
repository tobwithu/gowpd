package gowpd

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	Init()
	defer Destroy()
	os.Exit(m.Run())
}
func TestGetDevice(t *testing.T) {
	n := GetDeviceCount()
	if n == 0 {
		t.Errorf("No device")
	}
	for i := 0; i < n; i++ {
		if GetDeviceName(i) == "" || GetDeviceDescription(i) == "" {
			t.Errorf("%v - %v(%v)", i, GetDeviceName(i), GetDeviceDescription(i))
		}
	}
}
func TestChooseDevice(t *testing.T) {
	_, hr, _ := ChooseDevice(0)
	if hr < 0 {
		t.Errorf("ChooseDevice")
	}
}
