// +build windows

package gowpd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	CLIENT_NAME      = "gowpd"
	CLIENT_MAJOR_VER = 1
	CLIENT_MINOR_VER = 0
	CLIENT_REVISION  = 0

	MAX_PATH               = 260
	SECURITY_IMPERSONATION = 0x00020000
	GENERIC_READ           = 0x80000000
	STGC_DEFAULT           = 0

	PathSeparator = string(os.PathSeparator)
)

type GUID syscall.GUID

func CLSIDFromString(s string) *GUID {
	var id GUID
	s = strings.Replace(s, "-", "", -1)
	ui64, _ := strconv.ParseUint(s[:8], 16, 32)
	id.Data1 = uint32(ui64)
	ui64, _ = strconv.ParseUint(s[8:12], 16, 16)
	id.Data2 = uint16(ui64)
	ui64, _ = strconv.ParseUint(s[12:16], 16, 16)
	id.Data3 = uint16(ui64)
	ui64, _ = strconv.ParseUint(s[16:], 16, 64)
	binary.BigEndian.PutUint64(id.Data4[:], ui64)
	return &id
}

func (id GUID) String() string {
	var s string
	s = fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		id.Data1, id.Data2, id.Data3,
		id.Data4[0], id.Data4[1], id.Data4[2], id.Data4[3],
		id.Data4[4], id.Data4[5], id.Data4[6], id.Data4[7])
	return s
}

type PROPERTYKEY struct {
	Fmtid GUID
	Pid   uint32
}

var (
	ole                  = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx   = ole.NewProc("CoInitializeEx")
	procCoUninitialize   = ole.NewProc("CoUninitialize")
	procCoCreateInstance = ole.NewProc("CoCreateInstance")
	procCoTaskMemFree    = ole.NewProc("CoTaskMemFree")
	procPropVariantClear = ole.NewProc("PropVariantClear")
)

func handleError(ret uintptr, err syscall.Errno) (int32, error) {
	hr := int32(ret)
	if hr >= 0 {
		return hr, nil
	}
	if err == 0 {
		return hr, fmt.Errorf("Error:%0#x", ret)
	}
	return hr, err
}

func Syscall(trap, nargs, a1, a2, a3 uintptr) (int32, error) {
	ret, _, err := syscall.Syscall(trap, nargs, a1, a2, a3)
	return handleError(ret, err)
}

func Syscall6(trap, nargs, a1, a2, a3, a4, a5, a6 uintptr) (int32, error) {
	ret, _, err := syscall.Syscall6(trap, nargs, a1, a2, a3, a4, a5, a6)
	return handleError(ret, err)
}

func CoInitializeEx() (int32, error) {
	ret, _, err := procCoInitializeEx.Call(0, 0)
	hr := int32(ret)
	if hr >= 0 {
		err = nil
	}
	return hr, err
}

func CoUninitialize() {
	procCoUninitialize.Call()
}

func CoCreateInstance(clsId string, iid string, p interface{}) (int32, error) {
	ret, _, err := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(CLSIDFromString(clsId))),
		0,
		1,
		uintptr(unsafe.Pointer(CLSIDFromString(iid))),
		reflect.ValueOf(p).Pointer())
	hr := int32(ret)
	if hr >= 0 {
		err = nil
	}
	return hr, err
}

func CoTaskMemFree(p uintptr) {
	procCoTaskMemFree.Call(p)
}
func PropVariantClear(p *PROPVARIANT) {
	procPropVariantClear.Call(uintptr(unsafe.Pointer(p)))
}

type IUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

type IUnknown struct {
	vtbl *IUnknownVtbl
}

func (o *IUnknown) Vtable() *IUnknownVtbl {
	return o.vtbl
}
func (o *IUnknown) AddRef() (int32, error) {
	return Syscall(
		o.Vtable().AddRef,
		1,
		uintptr(unsafe.Pointer(o)),
		0,
		0)
}

func (o *IUnknown) Release() (int32, error) {
	hr, err := Syscall(
		o.Vtable().Release,
		1,
		uintptr(unsafe.Pointer(o)),
		0,
		0)
	if hr > 0 {
		fmt.Printf("Call Release() %p %v\n", o, hr)
	}
	return hr, err
}

type StreamReader struct {
	stream *IStream
}

func (o *StreamReader) Read(buf []byte) (int, error) {
	n, hr, err := o.stream.Read(buf, uint32(len(buf)))
	if n == 0 {
		return 0, io.EOF
	}
	if hr >= 0 {
		err = nil
	}
	return int(n), err
}

func (o *StreamReader) Close() error {
	o.stream.Release()
	return nil
}

type StreamWriter struct {
	stream *IPortableDeviceDataStream
}

func (o *StreamWriter) Write(buf []byte) (int, error) {
	n, hr, err := o.stream.Write(buf, uint32(len(buf)))
	if hr >= 0 {
		err = nil
	}
	return int(n), err
}

func (o *StreamWriter) Commit() (string, int32, error) {
	defer o.stream.Release()
	hr, err := o.stream.Commit(STGC_DEFAULT)
	if hr < 0 {
		return "", hr, err
	}
	return o.stream.GetObjectID()
}

func (o *StreamWriter) Close() error {
	defer o.stream.Release()
	_, err := o.stream.Commit(STGC_DEFAULT)
	return err
}

type BufReadCloser struct {
	reader *bufio.Reader
	closer io.Closer
}

func NewBufReadCloser(s io.ReadCloser, size int) *BufReadCloser {
	if size <= 0 {
		return &BufReadCloser{bufio.NewReader(s), s}
	}
	return &BufReadCloser{bufio.NewReaderSize(s, size), s}
}

func (o *BufReadCloser) Read(buf []byte) (int, error) {
	return o.reader.Read(buf)
}

func (o *BufReadCloser) Close() error {
	return o.closer.Close()
}

type BufWriteCloser struct {
	writer *bufio.Writer
	closer io.Closer
}

func NewBufWriteCloser(s io.WriteCloser, size int) *BufWriteCloser {
	if size <= 0 {
		return &BufWriteCloser{bufio.NewWriter(s), s}
	}
	return &BufWriteCloser{bufio.NewWriterSize(s, size), s}
}

func (o *BufWriteCloser) Write(buf []byte) (int, error) {
	return o.writer.Write(buf)
}
func (o *BufWriteCloser) Close() error {
	o.writer.Flush()
	return o.closer.Close()
}

func getClientInformation() (cInfo *IPortableDeviceValues) {
	hr, _ := CoCreateInstance(CLSID_PortableDeviceValues, IID_IPortableDeviceValues, &cInfo)
	if hr < 0 {
		return
	}
	cInfo.SetStringValue(WPD_CLIENT_NAME, CLIENT_NAME)
	cInfo.SetUnsignedIntegerValue(WPD_CLIENT_MAJOR_VERSION, CLIENT_MAJOR_VER)
	cInfo.SetUnsignedIntegerValue(WPD_CLIENT_MINOR_VERSION, CLIENT_MINOR_VER)
	cInfo.SetUnsignedIntegerValue(WPD_CLIENT_REVISION, CLIENT_REVISION)
	cInfo.SetUnsignedIntegerValue(WPD_CLIENT_SECURITY_QUALITY_OF_SERVICE, SECURITY_IMPERSONATION)
	return
}

func getPropertiesToRead() (keys *IPortableDeviceKeyCollection) {
	hr, _ := CoCreateInstance(CLSID_PortableDeviceKeyCollection, IID_PortableDeviceKeyCollection, &keys)
	if hr < 0 {
		return
	}
	keys.Add(WPD_OBJECT_PARENT_ID)
	keys.Add(WPD_OBJECT_CONTENT_TYPE)
	keys.Add(WPD_OBJECT_SIZE)
	keys.Add(WPD_OBJECT_ORIGINAL_FILE_NAME)
	keys.Add(WPD_OBJECT_DATE_MODIFIED)
	return
}

func getPropVariantCollection(id string) (*IPortableDevicePropVariantCollection, error) {
	var list *IPortableDevicePropVariantCollection
	hr, err := CoCreateInstance(CLSID_PortableDevicePropVariantCollection, IID_IPortableDevicePropVariantCollection, &list)
	if hr < 0 {
		return nil, err
	}
	var pv PROPVARIANT
	pv.Vt = VT_LPWSTR
	pt := syscall.StringToUTF16Ptr(id)
	pv.Val1 = uintptr(unsafe.Pointer(pt))
	list.Add(&pv)
	return list, nil
}

func ObjectFromFileInfo(path string, info os.FileInfo) *Object {
	var o Object
	o.Name = info.Name()
	o.Size = info.Size()
	o.ModTime = info.ModTime().Unix()
	o.IsDir = info.IsDir()
	o.Id = path
	return &o
}

func SetFileTime(path string, t int64) error {
	tm := time.Unix(t, 0)
	return os.Chtimes(path, tm, tm)
}

func CleanPath(path string) string {
	if "/" != PathSeparator {
		path = strings.ReplaceAll(path, "/", PathSeparator)
	}
	return strings.TrimRight(path, PathSeparator)
}
