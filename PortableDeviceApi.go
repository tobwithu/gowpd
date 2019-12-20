// +build windows

package gowpd

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const (
	CLSID_PortableDeviceManager               = "0af10cec-2ecd-4b92-9581-34f6ae0637f3"
	IID_IPortableDeviceManager                = "a1567595-4c2f-4574-a6fa-ecef917b9a40"
	CLSID_PortableDeviceValues                = "0c15d503-d017-47ce-9016-7b3f978721cc"
	IID_IPortableDeviceValues                 = "6848f6f2-3155-4f86-b6f5-263eeeab3143"
	CLSID_PortableDeviceFTM                   = "f7c0039a-4762-488a-b4b3-760ef9a1ba9b"
	IID_IPortableDevice                       = "625e2df8-6392-4cf0-9ad1-3cfa5f17775c"
	CLSID_PortableDeviceKeyCollection         = "de2d022d-2480-43be-97f0-d1fa2cf98f4f"
	IID_PortableDeviceKeyCollection           = "dada2357-e0ad-492e-98db-dd61c53ba353"
	CLSID_PortableDevicePropVariantCollection = "08a99e2f-6d6d-4b80-af5a-baf2bcbe4cb9"
	IID_IPortableDevicePropVariantCollection  = "89b2e422-4f1b-4316-bcef-a44afea83eb3"

	WPD_DEVICE_OBJECT_ID                = "DEVICE"
	STGM_READ                           = 0x00000000
	STGM_WRITE                          = 0x00000001
	STGM_CREATE                         = 0x00001000
	PORTABLE_DEVICE_DELETE_NO_RECURSION = 0

	NUM_OBJECTS_TO_REQUEST = 10
)

var (
	WPD_CLIENT_NAME                        = PROPERTYKEY{GUID{0x204D9F0C, 0x2292, 0x4080, [8]byte{0x9F, 0x42, 0x40, 0x66, 0x4E, 0x70, 0xF8, 0x59}}, 2}
	WPD_CLIENT_MAJOR_VERSION               = PROPERTYKEY{GUID{0x204D9F0C, 0x2292, 0x4080, [8]byte{0x9F, 0x42, 0x40, 0x66, 0x4E, 0x70, 0xF8, 0x59}}, 3}
	WPD_CLIENT_MINOR_VERSION               = PROPERTYKEY{GUID{0x204D9F0C, 0x2292, 0x4080, [8]byte{0x9F, 0x42, 0x40, 0x66, 0x4E, 0x70, 0xF8, 0x59}}, 4}
	WPD_CLIENT_REVISION                    = PROPERTYKEY{GUID{0x204D9F0C, 0x2292, 0x4080, [8]byte{0x9F, 0x42, 0x40, 0x66, 0x4E, 0x70, 0xF8, 0x59}}, 5}
	WPD_CLIENT_SECURITY_QUALITY_OF_SERVICE = PROPERTYKEY{GUID{0x204D9F0C, 0x2292, 0x4080, [8]byte{0x9F, 0x42, 0x40, 0x66, 0x4E, 0x70, 0xF8, 0x59}}, 8}
	WPD_CLIENT_DESIRED_ACCESS              = PROPERTYKEY{GUID{0x204D9F0C, 0x2292, 0x4080, [8]byte{0x9F, 0x42, 0x40, 0x66, 0x4E, 0x70, 0xF8, 0x59}}, 9}

	WPD_OBJECT_PARENT_ID                       = PROPERTYKEY{GUID{0xEF6B490D, 0x5CD8, 0x437A, [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}, 3}
	WPD_OBJECT_NAME                            = PROPERTYKEY{GUID{0xEF6B490D, 0x5CD8, 0x437A, [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}, 4}
	WPD_OBJECT_CONTENT_TYPE                    = PROPERTYKEY{GUID{0xEF6B490D, 0x5CD8, 0x437A, [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}, 7}
	WPD_OBJECT_SIZE                            = PROPERTYKEY{GUID{0xEF6B490D, 0x5CD8, 0x437A, [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}, 11}
	WPD_OBJECT_ORIGINAL_FILE_NAME              = PROPERTYKEY{GUID{0xEF6B490D, 0x5CD8, 0x437A, [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}, 12}
	WPD_OBJECT_DATE_CREATED                    = PROPERTYKEY{GUID{0xEF6B490D, 0x5CD8, 0x437A, [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}, 18}
	WPD_OBJECT_DATE_MODIFIED                   = PROPERTYKEY{GUID{0xEF6B490D, 0x5CD8, 0x437A, [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}, 19}
	WPD_PROPERTY_ATTRIBUTE_CAN_WRITE           = PROPERTYKEY{GUID{0xAB7943D8, 0x6332, 0x445F, [8]byte{0xA0, 0x0D, 0x8D, 0x5E, 0xF1, 0xE9, 0x6F, 0x37}}, 4}
	WPD_RESOURCE_DEFAULT                       = PROPERTYKEY{GUID{0xE81E79BE, 0x34F0, 0x41BF, [8]byte{0xB5, 0x3F, 0xF1, 0xA0, 0x6A, 0xE8, 0x78, 0x42}}, 0}
	WPD_COMMAND_OBJECT_MANAGEMENT_MOVE_OBJECTS = PROPERTYKEY{GUID{0xEF1E43DD, 0xA9ED, 0x4341, [8]byte{0x8B, 0xCC, 0x18, 0x61, 0x92, 0xAE, 0xA0, 0x89}}, 8}
	WPD_COMMAND_OBJECT_MANAGEMENT_COPY_OBJECTS = PROPERTYKEY{GUID{0xEF1E43DD, 0xA9ED, 0x4341, [8]byte{0x8B, 0xCC, 0x18, 0x61, 0x92, 0xAE, 0xA0, 0x89}}, 9}
	WPD_CONTENT_TYPE_FUNCTIONAL_OBJECT         = GUID{0x99ED0160, 0x17FF, 0x4C44, [8]byte{0x9D, 0x98, 0x1D, 0x7A, 0x6F, 0x94, 0x19, 0x21}}
	WPD_CONTENT_TYPE_FOLDER                    = GUID{0x27E2E392, 0xA111, 0x48E0, [8]byte{0xAB, 0x0C, 0xE1, 0x77, 0x05, 0xA0, 0x5F, 0x85}}
)

var (
	deviceIds []string
)

type IPortableDeviceManagerVtbl struct {
	IUnknownVtbl
	GetDevices            uintptr
	RefreshDeviceList     uintptr
	GetDeviceFriendlyName uintptr
	GetDeviceDescription  uintptr
	GetDeviceManufacturer uintptr
	GetDeviceProperty     uintptr
	GetPrivateDevices     uintptr
}

type IPortableDeviceManager struct {
	IUnknown
}

func (o *IPortableDeviceManager) Vtable() *IPortableDeviceManagerVtbl {
	return (*IPortableDeviceManagerVtbl)(unsafe.Pointer(o.vtbl))
}

func NewIPortableDeviceManager() (*IPortableDeviceManager, int32, error) {
	var deviceManager *IPortableDeviceManager
	hr, err := CoCreateInstance(CLSID_PortableDeviceManager, IID_IPortableDeviceManager, &deviceManager)
	return deviceManager, hr, err
}

func (o *IPortableDeviceManager) GetDevices() (count uint32, hr int32, err error) {
	hr, err = Syscall(
		o.Vtable().GetDevices,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(0),
		uintptr(unsafe.Pointer(&count)))
	deviceIds = make([]string, count)
	if count > 0 {
		apwstr := make([]uintptr, count)
		hr, err = Syscall(
			o.Vtable().GetDevices,
			3,
			uintptr(unsafe.Pointer(o)),
			uintptr(unsafe.Pointer(&apwstr[0])),
			uintptr(unsafe.Pointer(&count)))
		for i := uint32(0); i < count; i++ {
			if apwstr[i] != 0 {
				deviceIds[i] = syscall.UTF16ToString((*[MAX_PATH]uint16)(unsafe.Pointer(apwstr[i]))[:])
				CoTaskMemFree(apwstr[i])
			}
		}
	}
	return count, hr, err
}

func (o *IPortableDeviceManager) RefreshDeviceList() int32 {
	hr, _ := Syscall(
		o.Vtable().RefreshDeviceList,
		1,
		uintptr(unsafe.Pointer(o)),
		0,
		0)
	return hr
}

func (o *IPortableDeviceManager) getDeviceString(cmd uintptr, id int) (string, int32, error) {
	if id >= len(deviceIds) {
		return "", -1, nil
	}
	len := 0
	hr, err := Syscall6(
		cmd,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(deviceIds[id]))),
		uintptr(0),
		uintptr(unsafe.Pointer(&len)),
		0, 0)
	awchar := make([]uint16, len)
	hr, err = Syscall6(
		cmd,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(deviceIds[id]))),
		uintptr(unsafe.Pointer(&awchar[0])),
		uintptr(unsafe.Pointer(&len)),
		0, 0)
	str := syscall.UTF16ToString(awchar)
	return str, hr, err
}

func (o *IPortableDeviceManager) GetDeviceFriendlyName(id int) (string, int32, error) {
	return o.getDeviceString(o.Vtable().GetDeviceFriendlyName, id)
}

func (o *IPortableDeviceManager) GetDeviceDescription(id int) (string, int32, error) {
	return o.getDeviceString(o.Vtable().GetDeviceDescription, id)
}

func (o *IPortableDeviceManager) GetDeviceManufacturer(id int) (string, int32, error) {
	return o.getDeviceString(o.Vtable().GetDeviceManufacturer, id)
}

func (o *IPortableDeviceManager) ChooseDevice(id int, cInfo *IPortableDeviceValues) (*IPortableDevice, int32, error) {
	if id >= len(deviceIds) {
		return nil, -1, fmt.Errorf("Invalied index : %v", id)
	}
	var device *IPortableDevice
	hr, err := CoCreateInstance(CLSID_PortableDeviceFTM, IID_IPortableDevice, &device)
	if hr < 0 {
		return device, hr, err
	}

	hr, err = device.Open(deviceIds[id], cInfo)
	if hr == -2147024891 { //E_ACCESSDENIED 0x80070005
		cInfo.SetUnsignedIntegerValue(WPD_CLIENT_DESIRED_ACCESS, GENERIC_READ)
		hr, err = device.Open(deviceIds[id], cInfo)
		if hr < 0 {
			device.Release()
			device = nil
		}
	}
	return device, hr, err
}

type IPortableDeviceValuesVtbl struct {
	IUnknownVtbl
	GetCount                                     uintptr
	GetAt                                        uintptr
	SetValue                                     uintptr
	GetValue                                     uintptr
	SetStringValue                               uintptr
	GetStringValue                               uintptr
	SetUnsignedIntegerValue                      uintptr
	GetUnsignedIntegerValue                      uintptr
	SetSignedIntegerValue                        uintptr
	GetSignedIntegerValue                        uintptr
	SetUnsignedLargeIntegerValue                 uintptr
	GetUnsignedLargeIntegerValue                 uintptr
	SetSignedLargeIntegerValue                   uintptr
	GetSignedLargeIntegerValue                   uintptr
	SetFloatValue                                uintptr
	GetFloatValue                                uintptr
	SetErrorValue                                uintptr
	GetErrorValue                                uintptr
	SetKeyValue                                  uintptr
	GetKeyValue                                  uintptr
	SetBoolValue                                 uintptr
	GetBoolValue                                 uintptr
	SetIUnknownValue                             uintptr
	GetIUnknownValue                             uintptr
	SetGuidValue                                 uintptr
	GetGuidValue                                 uintptr
	SetBufferValue                               uintptr
	GetBufferValue                               uintptr
	SetIPortableDeviceValuesValue                uintptr
	GetIPortableDeviceValuesValue                uintptr
	SetIPortableDevicePropVariantCollectionValue uintptr
	GetIPortableDevicePropVariantCollectionValue uintptr
	SetIPortableDeviceKeyCollectionValue         uintptr
	GetIPortableDeviceKeyCollectionValue         uintptr
	SetIPortableDeviceValuesCollectionValue      uintptr
	IPortableDeviceValuesCollection              uintptr
	GetIPortableDeviceValuesCollectionValue      uintptr
	RemoveValue                                  uintptr
	CopyValuesFromPropertyStore                  uintptr
	CopyValuesToPropertyStore                    uintptr
	Clear                                        uintptr
}
type IPortableDeviceValues struct {
	IUnknown
}

func (o *IPortableDeviceValues) Vtable() *IPortableDeviceValuesVtbl {
	return (*IPortableDeviceValuesVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *IPortableDeviceValues) SetValue(key PROPERTYKEY, val *PROPVARIANT) (int32, error) {
	return Syscall(
		o.Vtable().SetValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(val)))
}
func (o *IPortableDeviceValues) GetValue(key PROPERTYKEY) (*PROPVARIANT, int32, error) {
	var val PROPVARIANT
	hr, err := Syscall(
		o.Vtable().GetValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&val)))
	return &val, hr, err
}

func (o *IPortableDeviceValues) SetStringValue(key PROPERTYKEY, val string) (int32, error) {
	return Syscall(
		o.Vtable().SetStringValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(val))))
}
func (o *IPortableDeviceValues) GetStringValue(key PROPERTYKEY) (string, int32, error) {
	var pwchar uintptr
	hr, err := Syscall(
		o.Vtable().GetStringValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&pwchar)))
	var str string
	if pwchar != 0 {
		str = syscall.UTF16ToString((*[MAX_PATH]uint16)(unsafe.Pointer(pwchar))[:])
		CoTaskMemFree(pwchar)
	}
	return str, hr, err
}
func (o *IPortableDeviceValues) SetUnsignedIntegerValue(key PROPERTYKEY, val uint32) (int32, error) {
	return Syscall(
		o.Vtable().SetUnsignedIntegerValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(val))
}
func (o *IPortableDeviceValues) GetUnsignedIntegerValue(key PROPERTYKEY) (uint32, int32, error) {
	var val uint32
	hr, err := Syscall(
		o.Vtable().GetUnsignedIntegerValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&val)))
	return val, hr, err
}
func (o *IPortableDeviceValues) SetUnsignedLargeIntegerValue(key PROPERTYKEY, val uint64) (int32, error) {
	return Syscall(
		o.Vtable().SetUnsignedLargeIntegerValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(val))
}
func (o *IPortableDeviceValues) GetUnsignedLargeIntegerValue(key PROPERTYKEY) (uint64, int32, error) {
	var val uint64
	hr, err := Syscall(
		o.Vtable().GetUnsignedLargeIntegerValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&val)))
	return val, hr, err
}
func (o *IPortableDeviceValues) SetGuidValue(key PROPERTYKEY, val GUID) (int32, error) {
	return Syscall(
		o.Vtable().SetGuidValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&val)))
}
func (o *IPortableDeviceValues) GetGuidValue(key PROPERTYKEY) (GUID, int32, error) {
	var val GUID
	hr, err := Syscall(
		o.Vtable().GetGuidValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&val)))
	return val, hr, err
}

func (o *IPortableDeviceValues) SetBoolValue(key PROPERTYKEY, b bool) (int32, error) {
	var val uintptr = 0
	if b {
		val = 1
	}
	return Syscall(
		o.Vtable().SetBoolValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(val))
}

func (o *IPortableDeviceValues) GetBoolValue(key PROPERTYKEY) (bool, int32, error) {
	var val bool
	hr, err := Syscall(
		o.Vtable().GetBoolValue,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&val)))
	return val, hr, err
}

func (o *IPortableDeviceValues) SetUnixTimeValue(key PROPERTYKEY, t int64) (int32, error) {
	var val PROPVARIANT
	val.Vt = VT_DATE
	vtime := UnixTimeToVariantTime(t)
	val.Val1 = *((*uintptr)(unsafe.Pointer(&vtime)))
	return o.SetValue(key, &val)
}

func (o *IPortableDeviceValues) GetUnixTimeValue(key PROPERTYKEY) (t int64, hr int32, err error) {
	val, hr, err := o.GetValue(key)
	if hr < 0 || val.Vt != VT_DATE {
		return
	}
	t = VariantTimeToUnixTime(*((*float64)(unsafe.Pointer(&val.Val1))))
	PropVariantClear(val)
	return
}

type IPortableDeviceVtbl struct {
	IUnknownVtbl
	Open           uintptr
	SendCommand    uintptr
	Content        uintptr
	Capabilities   uintptr
	Cancel         uintptr
	Close          uintptr
	Advise         uintptr
	Unadvise       uintptr
	GetPnPDeviceID uintptr
}
type IPortableDevice struct {
	IUnknown
}

func (o *IPortableDevice) Vtable() *IPortableDeviceVtbl {
	return (*IPortableDeviceVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *IPortableDevice) Open(deviceId string, cInfo *IPortableDeviceValues) (int32, error) {
	return Syscall(
		o.Vtable().Open,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(deviceId))),
		uintptr(unsafe.Pointer(cInfo)))
}

func (o *IPortableDevice) Content() (*IPortableDeviceContent, int32, error) {
	var content *IPortableDeviceContent
	hr, err := Syscall(
		o.Vtable().Content,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&content)),
		0)
	return content, hr, err
}

func (o *IPortableDevice) Capabilities() (*IPortableDeviceCapabilities, int32, error) {
	var capa *IPortableDeviceCapabilities
	hr, err := Syscall(
		o.Vtable().Capabilities,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&capa)),
		0)
	return capa, hr, err
}

type IPortableDeviceContentVtbl struct {
	IUnknownVtbl
	EnumObjects                         uintptr
	Properties                          uintptr
	Transfer                            uintptr
	CreateObjectWithPropertiesOnly      uintptr
	CreateObjectWithPropertiesAndData   uintptr
	Delete                              uintptr
	GetObjectIDsFromPersistentUniqueIDs uintptr
	Cancel                              uintptr
	Move                                uintptr
	Copy                                uintptr
}
type IPortableDeviceContent struct {
	IUnknown
}

func (o *IPortableDeviceContent) Vtable() *IPortableDeviceContentVtbl {
	return (*IPortableDeviceContentVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *IPortableDeviceContent) EnumObjects(id string) (*IEnumPortableDeviceObjectIDs, int32, error) {
	var en *IEnumPortableDeviceObjectIDs
	hr, err := Syscall6(
		o.Vtable().EnumObjects,
		5,
		uintptr(unsafe.Pointer(o)),
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(id))),
		0,
		uintptr(unsafe.Pointer(&en)),
		0)
	return en, hr, err
}

func (o *IPortableDeviceContent) Properties() (*IPortableDeviceProperties, int32, error) {
	var properties *IPortableDeviceProperties
	hr, err := Syscall(
		o.Vtable().Properties,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&properties)),
		0)
	return properties, hr, err
}

func (o *IPortableDeviceContent) Transfer() (*IPortableDeviceResources, int32, error) {
	var resources *IPortableDeviceResources
	hr, err := Syscall(
		o.Vtable().Transfer,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&resources)),
		0)
	return resources, hr, err
}

func (o *IPortableDeviceContent) CreateObjectWithPropertiesOnly(properties *IPortableDeviceValues) (string, int32, error) {
	var pwstr uintptr
	hr, err := Syscall(
		o.Vtable().CreateObjectWithPropertiesOnly,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(properties)),
		uintptr(unsafe.Pointer(&pwstr)))
	var id string
	if hr >= 0 {
		id = syscall.UTF16ToString((*[MAX_PATH]uint16)(unsafe.Pointer(pwstr))[:])
		CoTaskMemFree(pwstr)
	}
	return id, hr, err
}

func (o *IPortableDeviceContent) CreateObjectWithPropertiesAndData(properties *IPortableDeviceValues) (*IPortableDeviceDataStream, uint32, int32, error) {
	var stream *IPortableDeviceDataStream
	var transferSize uint32
	hr, err := Syscall6(
		o.Vtable().CreateObjectWithPropertiesAndData,
		5,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(properties)),
		uintptr(unsafe.Pointer(&stream)),
		uintptr(unsafe.Pointer(&transferSize)),
		0, 0)
	return stream, transferSize, hr, err
}

func (o *IPortableDeviceContent) Delete(option int, list *IPortableDevicePropVariantCollection) (*IPortableDevicePropVariantCollection, int32, error) {
	hr, err := Syscall6(
		o.Vtable().Delete,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(option),
		uintptr(unsafe.Pointer(list)),
		0,
		0, 0)
	return nil, hr, err
}

func (o *IPortableDeviceContent) Copy(list *IPortableDevicePropVariantCollection, id string) (*IPortableDevicePropVariantCollection, int32, error) {
	hr, err := Syscall6(
		o.Vtable().Copy,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(list)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(id))),
		0,
		0, 0)
	return nil, hr, err
}

type IEnumPortableDeviceObjectIDsVtbl struct {
	IUnknownVtbl
	Next   uintptr
	Skip   uintptr
	Reset  uintptr
	Clone  uintptr
	Cancel uintptr
}

type IEnumPortableDeviceObjectIDs struct {
	IUnknown
}

func (o *IEnumPortableDeviceObjectIDs) Vtable() *IEnumPortableDeviceObjectIDsVtbl {
	return (*IEnumPortableDeviceObjectIDsVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *IEnumPortableDeviceObjectIDs) Next() ([]string, int32, error) {
	apwstr := make([]uintptr, NUM_OBJECTS_TO_REQUEST)
	var len uint32
	hr, err := Syscall6(
		o.Vtable().Next,
		4,
		uintptr(unsafe.Pointer(o)),
		NUM_OBJECTS_TO_REQUEST,
		uintptr(unsafe.Pointer(&apwstr[0])),
		uintptr(unsafe.Pointer(&len)), 0, 0)
	ids := make([]string, len)
	for i := uint32(0); i < len; i++ {
		if apwstr[i] != 0 {
			ids[i] = syscall.UTF16ToString((*[MAX_PATH]uint16)(unsafe.Pointer(apwstr[i]))[:])
			CoTaskMemFree(apwstr[i])
		}
	}
	return ids, hr, err
}

type IPortableDevicePropertiesVtbl struct {
	IUnknownVtbl
	GetSupportedProperties uintptr
	GetPropertyAttributes  uintptr
	GetValues              uintptr
	SetValues              uintptr
	Delete                 uintptr
	Cancel                 uintptr
}

type IPortableDeviceProperties struct {
	IUnknown
}

func (o *IPortableDeviceProperties) Vtable() *IPortableDevicePropertiesVtbl {
	return (*IPortableDevicePropertiesVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *IPortableDeviceProperties) GetValues(id string, keys *IPortableDeviceKeyCollection) (*IPortableDeviceValues, int32, error) {
	var v *IPortableDeviceValues
	hr, err := CoCreateInstance(CLSID_PortableDeviceValues, IID_IPortableDeviceValues, &v)
	if hr < 0 {
		return v, hr, err
	}
	hr, err = Syscall6(
		o.Vtable().GetValues,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(id))),
		uintptr(unsafe.Pointer(keys)),
		uintptr(unsafe.Pointer(&v)), 0, 0)
	return v, hr, err
}

func (o *IPortableDeviceProperties) GetPropertyAttributes(id string, key PROPERTYKEY) (*IPortableDeviceValues, int32, error) {
	var v *IPortableDeviceValues
	hr, err := CoCreateInstance(CLSID_PortableDeviceValues, IID_IPortableDeviceValues, &v)
	if hr < 0 {
		return v, hr, err
	}
	hr, err = Syscall6(
		o.Vtable().GetPropertyAttributes,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(id))),
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&v)), 0, 0)
	return v, hr, err
}

type IPortableDeviceKeyCollectionVtbl struct {
	IUnknownVtbl
	GetCount uintptr
	GetAt    uintptr
	Add      uintptr
	Clear    uintptr
	RemoveAt uintptr
}

type IPortableDeviceKeyCollection struct {
	IUnknown
}

func (o *IPortableDeviceKeyCollection) Vtable() *IPortableDeviceKeyCollectionVtbl {
	return (*IPortableDeviceKeyCollectionVtbl)(unsafe.Pointer(o.vtbl))
}
func (o *IPortableDeviceKeyCollection) GetCount() (int, int32, error) {
	var val int
	hr, err := Syscall(
		o.Vtable().GetCount,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&val)),
		0)
	return val, hr, err
}
func (o *IPortableDeviceKeyCollection) GetAt(ind int) (PROPERTYKEY, int32, error) {
	var val PROPERTYKEY
	hr, err := Syscall(
		o.Vtable().GetAt,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(ind),
		uintptr(unsafe.Pointer(&val)))
	return val, hr, err
}

func (o *IPortableDeviceKeyCollection) Add(key PROPERTYKEY) (int32, error) {
	hr, err := Syscall(
		o.Vtable().Add,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&key)),
		0)
	return hr, err
}

type IPortableDeviceResourcesVtbl struct {
	IUnknownVtbl
	GetSupportedResources uintptr
	GetResourceAttributes uintptr
	GetStream             uintptr
	Delete                uintptr
	Cancel                uintptr
	CreateResource        uintptr
}

type IPortableDeviceResources struct {
	IUnknown
}

func (o *IPortableDeviceResources) Vtable() *IPortableDeviceResourcesVtbl {
	return (*IPortableDeviceResourcesVtbl)(unsafe.Pointer(o.vtbl))
}

type ISequentialStreamVtbl struct {
	IUnknownVtbl
	Read  uintptr
	Write uintptr
}

type ISequentialStream struct {
	IUnknown
}

func (o *ISequentialStream) Vtable() *ISequentialStreamVtbl {
	return (*ISequentialStreamVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *ISequentialStream) Read(buf []byte, size uint32) (uint32, int32, error) {
	var read uint32
	hr, err := Syscall6(
		o.Vtable().Read,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(size),
		uintptr(unsafe.Pointer(&read)), 0, 0)
	return read, hr, err
}

func (o *ISequentialStream) Write(buf []byte, size uint32) (uint32, int32, error) {
	var written uint32
	hr, err := Syscall6(
		o.Vtable().Write,
		4,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(size),
		uintptr(unsafe.Pointer(&written)), 0, 0)
	return written, hr, err
}

type IStreamVtbl struct {
	ISequentialStreamVtbl
	Seek         uintptr
	SetSize      uintptr
	CopyTo       uintptr
	Commit       uintptr
	Revert       uintptr
	LockRegion   uintptr
	UnlockRegion uintptr
	Stat         uintptr
	Clone        uintptr
}
type IStream struct {
	ISequentialStream
}

func (o *IStream) Vtable() *IStreamVtbl {
	return (*IStreamVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *IStream) Commit(flag uint32) (int32, error) {
	return Syscall(
		o.Vtable().Commit,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(flag),
		0)
}

func (o *IPortableDeviceResources) GetStream(id string) (*IStream, uint32, int32, error) {
	var stream *IStream
	var transferSize uint32
	hr, err := Syscall6(
		o.Vtable().GetStream,
		6,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(id))),
		uintptr(unsafe.Pointer(&WPD_RESOURCE_DEFAULT)),
		STGM_READ,
		uintptr(unsafe.Pointer(&transferSize)),
		uintptr(unsafe.Pointer(&stream)))
	return stream, transferSize, hr, err
}

type IPortableDeviceDataStreamVtbl struct {
	IStreamVtbl
	GetObjectID uintptr
	Cancel      uintptr
}

type IPortableDeviceDataStream struct {
	IStream
}

func (o *IPortableDeviceDataStream) Vtable() *IPortableDeviceDataStreamVtbl {
	return (*IPortableDeviceDataStreamVtbl)(unsafe.Pointer(o.vtbl))
}

func (o *IPortableDeviceDataStream) GetObjectID() (string, int32, error) {
	var pwstr uintptr
	hr, err := Syscall(
		o.Vtable().GetObjectID,
		3,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&pwstr)),
		0)
	if hr < 0 {
		return "", hr, err
	}
	id := syscall.UTF16ToString((*[MAX_PATH]uint16)(unsafe.Pointer(pwstr))[:])
	CoTaskMemFree(pwstr)
	return id, hr, err
}

const (
	VT_DATE   = 7
	VT_LPWSTR = 31
)

type PROPVARIANT struct {
	Vt        uint16
	Reserved1 uint16
	Reserved2 uint16
	Reserved3 uint16
	Val1      uintptr
	Val2      int64
}

func VariantTimeToUnixTime(vtime float64) int64 {
	_, offset := time.Now().Zone()
	return int64((vtime-25569)*86400+0.5) - int64(offset)
}
func UnixTimeToVariantTime(utime int64) float64 {
	_, offset := time.Now().Zone()
	return float64(utime+int64(offset))/86400 + 25569
}

type IPortableDevicePropVariantCollectionVtbl struct {
	IUnknownVtbl
	GetCount   uintptr
	GetAt      uintptr
	Add        uintptr
	GetType    uintptr
	ChangeType uintptr
	Clear      uintptr
	RemoveAt   uintptr
}
type IPortableDevicePropVariantCollection struct {
	IUnknown
}

func (o *IPortableDevicePropVariantCollection) Vtable() *IPortableDevicePropVariantCollectionVtbl {
	return (*IPortableDevicePropVariantCollectionVtbl)(unsafe.Pointer(o.vtbl))
}
func (o *IPortableDevicePropVariantCollection) Add(pv *PROPVARIANT) (int32, error) {
	return Syscall(
		o.Vtable().Add,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(pv)),
		0)
}

type IPortableDeviceCapabilitiesVtbl struct {
	IUnknownVtbl
	GetSupportedCommands         uintptr
	GetCommandOptions            uintptr
	GetFunctionalCategories      uintptr
	GetFunctionalObjects         uintptr
	GetSupportedContentTypes     uintptr
	GetSupportedFormats          uintptr
	GetSupportedFormatProperties uintptr
	GetFixedPropertyAttributes   uintptr
	Cancel                       uintptr
	GetSupportedEvents           uintptr
	GetEventOptions              uintptr
}
type IPortableDeviceCapabilities struct {
	IUnknown
}

func (o *IPortableDeviceCapabilities) Vtable() *IPortableDeviceCapabilitiesVtbl {
	return (*IPortableDeviceCapabilitiesVtbl)(unsafe.Pointer(o.vtbl))
}
func (o *IPortableDeviceCapabilities) GetSupportedCommands() (*IPortableDeviceKeyCollection, int32, error) {
	var col *IPortableDeviceKeyCollection
	hr, err := Syscall(
		o.Vtable().GetSupportedCommands,
		2,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&col)),
		0)
	return col, hr, err
}
