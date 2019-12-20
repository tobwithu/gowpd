package gowpd

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	err := Init()
	defer Destroy()
	if err != nil {
		panic(err)
	}
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
	_, err := ChooseDevice(0)
	if err != nil {
		t.Errorf("%v", err)
	}
}
