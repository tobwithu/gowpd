// +build windows

package gowpd

import (
	"io"
	"os"
	"strings"
)

var (
	deviceManager *IPortableDeviceManager
)

type Device struct {
	device     *IPortableDevice
	content    *IPortableDeviceContent
	properties *IPortableDeviceProperties
	keys       *IPortableDeviceKeyCollection
	resources  *IPortableDeviceResources
	CanCopy    bool
}

type ObjectInfo struct {
	ModTime int64
	Size    int64
	IsDir   bool
}

type Object struct {
	ObjectInfo
	ChildCount int

	Id          string
	ParentId    string
	Name        string
	ContentType GUID
}

func Init() error {
	_, err := CoInitializeEx()
	if err != nil {
		return err
	}
	deviceManager, _, err = NewIPortableDeviceManager()
	if err != nil {
		return err
	}
	_, _, err = deviceManager.GetDevices()
	return err
}

func Destroy() {
	deviceManager.Release()
	CoUninitialize()
}

func GetDeviceCount() int {
	return len(deviceIds)
}

func GetDeviceName(id int) string {
	s, _, _ := deviceManager.GetDeviceFriendlyName(id)
	return s
}

func GetDeviceDescription(id int) string {
	s, _, _ := deviceManager.GetDeviceDescription(id)
	return strings.TrimRight(s, " ")
}

func GetDeviceManufacturer(id int) string {
	s, _, _ := deviceManager.GetDeviceManufacturer(id)
	return s
}

func GetDeviceId(name string) int {
	for i := 0; i < len(deviceIds); i++ {
		if name == GetDeviceName(i) {
			return i
		}
		if name == GetDeviceDescription(i) {
			return i
		}
	}
	return -1
}

func ChooseDevice(id int) (d *Device, err error) {
	d = &Device{}
	cInfo := getClientInformation()
	defer cInfo.Release()
	d.device, _, err = deviceManager.ChooseDevice(id, cInfo)
	if err != nil {
		return
	}
	d.content, _, err = d.device.Content()
	if err != nil {
		return
	}
	d.properties, _, err = d.content.Properties()
	if err != nil {
		return
	}
	d.resources, _, err = d.content.Transfer()
	if err != nil {
		return
	}
	d.CanCopy = d.SupportsCommand(WPD_COMMAND_OBJECT_MANAGEMENT_COPY_OBJECTS)
	return
}

func (d *Device) Release() {
	d.resources.Release()
	d.properties.Release()
	d.content.Release()
	d.device.Release()
}

func (d *Device) GetObject(id string) (o *Object, err error) {
	var v *IPortableDeviceValues
	v, _, err = d.properties.GetValues(id, d.keys)
	defer v.Release()
	if err != nil {
		return
	}
	o = &Object{}
	o.Id = id
	o.ParentId, _, err = v.GetStringValue(WPD_OBJECT_PARENT_ID)
	o.Name, _, err = v.GetStringValue(WPD_OBJECT_ORIGINAL_FILE_NAME)
	var size uint64
	size, _, err = v.GetUnsignedLargeIntegerValue(WPD_OBJECT_SIZE)
	o.Size = int64(size)
	o.ModTime, _, err = v.GetUnixTimeValue(WPD_OBJECT_DATE_MODIFIED)
	o.ContentType, _, err = v.GetGuidValue(WPD_OBJECT_CONTENT_TYPE)
	o.IsDir = o.ContentType == WPD_CONTENT_TYPE_FUNCTIONAL_OBJECT || o.ContentType == WPD_CONTENT_TYPE_FOLDER
	return
}

func (d *Device) GetChildIds(id string) (ids []string, err error) {
	var enum *IEnumPortableDeviceObjectIDs
	enum, _, err = d.content.EnumObjects(id)
	if err != nil {
		return
	}
	defer enum.Release()

	for {
		var ar []string
		ar, _, err = enum.Next()
		ids = append(ids, ar...)
		if len(ar) < NUM_OBJECTS_TO_REQUEST {
			break
		}
	}
	return
}

func (d *Device) GetChildObjects(id string) (ar []*Object, err error) {
	ids, err := d.GetChildIds(id)
	if err != nil {
		return
	}
	ar = make([]*Object, len(ids))
	for i, id := range ids {
		var o *Object
		o, err = d.GetObject(id)
		ar[i] = o
	}
	return
}

func (d *Device) findObject(path string, id string, curPath string) *Object {
	objs, _ := d.GetChildObjects(id)
	for _, o := range objs {
		newPath := curPath + o.Name + PathSeparator
		if path == newPath {
			return o
		} else if strings.Index(path, newPath) == 0 {
			if o.IsDir {
				rs := d.findObject(path, o.Id, newPath)
				if rs != nil {
					return rs
				}
			}
		}
	}
	return nil
}

func (d *Device) FindObject(path string) *Object {
	path = CleanPath(path) + PathSeparator
	obj := d.findObject(path, WPD_DEVICE_OBJECT_ID, "")
	return obj
}

func (d *Device) GetReader(id string) (*BufReadCloser, error) {
	stream, size, _, err := d.resources.GetStream(id)
	if err != nil {
		return nil, err
	}
	return NewBufReadCloser(&StreamReader{stream}, int(size)), nil
}
func (d *Device) CopyFromDevice(dst string, id string) (int64, error) {
	reader, err := d.GetReader(id)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	f, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	writer := NewBufWriteCloser(f, 0)
	defer writer.Close()
	return io.Copy(writer, reader)
}
func (d *Device) CopyObjectFromDevice(dst string, obj *Object) (int64, error) {
	written, err := d.CopyFromDevice(dst, obj.Id)
	if err != nil {
		return 0, err
	}
	return written, SetFileTime(dst, obj.ModTime)
}

func (d *Device) CopyToDevice(parentId string, src string) (int64, error) {
	info, err := os.Lstat(src)
	if info == nil || info.IsDir() {
		return 0, err
	}
	o := ObjectFromFileInfo(src, info)
	f, err := os.Open(o.Id)
	if err != nil {
		return 0, err
	}
	reader := NewBufReadCloser(f, 0)
	defer reader.Close()
	return d.CopyObjectToDevice(parentId, reader, o)
}

func (d *Device) CopyObjectToDevice(parentId string, src io.Reader, obj *Object) (int64, error) {
	var prop *IPortableDeviceValues
	_, err := CoCreateInstance(CLSID_PortableDeviceValues, IID_IPortableDeviceValues, &prop)
	if err != nil {
		return 0, err
	}
	prop.SetStringValue(WPD_OBJECT_PARENT_ID, parentId)
	ind := strings.Index(obj.Name, ".")
	name := obj.Name
	if ind > 0 {
		name = name[0:ind]
	}
	prop.SetStringValue(WPD_OBJECT_NAME, name)
	prop.SetStringValue(WPD_OBJECT_ORIGINAL_FILE_NAME, obj.Name)
	prop.SetUnsignedLargeIntegerValue(WPD_OBJECT_SIZE, uint64(obj.Size))
	prop.SetUnixTimeValue(WPD_OBJECT_DATE_MODIFIED, obj.ModTime)
	defer prop.Release()
	stream, size, _, err := d.content.CreateObjectWithPropertiesAndData(prop)
	if err != nil {
		return 0, err
	}

	writer := NewBufWriteCloser(&StreamWriter{stream}, int(size))
	n, err := io.Copy(writer, src)
	return n, writer.Close()
}

func (d *Device) Delete(id string) error {
	list, err := getPropVariantCollection(id)
	if err != nil {
		return err
	}
	defer list.Release()
	_, _, err = d.content.Delete(PORTABLE_DEVICE_DELETE_NO_RECURSION, list)
	return err
}

func (d *Device) SupportsCommand(cmd PROPERTYKEY) bool {
	capa, _, err := d.device.Capabilities()
	if err != nil {
		return false
	}
	defer capa.Release()
	cmds, _, err := capa.GetSupportedCommands()
	if err != nil {
		return false
	}
	defer cmds.Release()

	n, _, err := cmds.GetCount()
	for i := 0; i < n; i++ {
		c, _, _ := cmds.GetAt(i)
		if c == cmd {
			return true
		}
	}
	return false
}

func (d *Device) Copy(parentId string, id string) error {
	list, err := getPropVariantCollection(id)
	if err != nil {
		return err
	}
	defer list.Release()
	_, _, err = d.content.Copy(list, parentId)
	return err
}

func (d *Device) CreateFolder(parentId string, name string) (string, error) {
	var prop *IPortableDeviceValues
	_, err := CoCreateInstance(CLSID_PortableDeviceValues, IID_IPortableDeviceValues, &prop)
	if err != nil {
		return "", err
	}
	prop.SetStringValue(WPD_OBJECT_PARENT_ID, parentId)
	prop.SetStringValue(WPD_OBJECT_NAME, name)
	prop.SetStringValue(WPD_OBJECT_ORIGINAL_FILE_NAME, name)
	prop.SetGuidValue(WPD_OBJECT_CONTENT_TYPE, WPD_CONTENT_TYPE_FOLDER)
	defer prop.Release()

	id, _, err := d.content.CreateObjectWithPropertiesOnly(prop)
	return id, err
}
