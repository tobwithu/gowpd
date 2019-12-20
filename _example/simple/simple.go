package main

import (
	"fmt"
	"github.com/tobwithu/gowpd"
)

func main() {
	err := gowpd.Init()
	defer gowpd.Destroy()
	if err != nil {
		panic(err)
	}
	n := gowpd.GetDeviceCount()
	for i := 0; i < n; i++ {
		fmt.Printf("%v - %v (%v)\n", i, gowpd.GetDeviceName(i), gowpd.GetDeviceDescription(i))
	}
	d, err := gowpd.ChooseDevice(0)
	if err == nil {
		o := d.FindObject(gowpd.PathSeparator) //find root object
		if o != nil {
			fmt.Printf(o.Id)
		}
	}
}
