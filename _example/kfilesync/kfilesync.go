package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/tobwithu/gowpd"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
    "strings"
	"strconv"
)

const (
	ST_NEW = iota
	ST_NEWER
	ST_SAME
	ST_OLDER
	ST_NOT_EXIST

	LIST_FILENAME      = ".kfilesync"
	SYSTEM_VOLUME_INFO = "System Volume Information"
)

var (
	MTP_ID   = regexp.MustCompile("(?i)^MTP(\\d):(\\S*)")
	MTP_NAME = regexp.MustCompile("(^.{2,}?):(\\S*)")

	deviceCount      int
	src, dst         *FileManager
	srcList, dstList map[string]*gowpd.Object
)

func help() {
	fmt.Println("kfilesync src dst [mode]\n")
	fmt.Println("  src\tSource folder")
	fmt.Println("\t    ex) MTP0:\\DCIM  - DCIM folder of MTP device with id = 0")
	fmt.Println("  dst\tDestination folder")
	fmt.Println("  mode\tSync mode")
	fmt.Println("\t+  Copy new files from source folder (default)")
	fmt.Println("\t0  Init sync. Copy no files and just make list of files in source folder.")
	fmt.Println("\t   Current files in source folder will not be copied in next sync.")
	fmt.Println("\t-  Delete files which is not in source folder.")
	fmt.Println("\t=  Make destination folder equal to source folder.")
	fmt.Println("\t?  Show differences only.")
}

type PathInfo struct {
	deviceId string
	path     string
}

type FileManager struct {
	PathInfo
	device *gowpd.Device
}

func getMtpFileManager(id int, path string, relPath string) (*FileManager, error) {
	var fm FileManager
	fm.deviceId = string(id)
	fm.path = relPath
	if fm.path == "" {
		fm.path = gowpd.PathSeparator
	}
	fm.device, _ = gowpd.ChooseDevice(id)
	o := fm.device.FindObject(fm.path)
	if o == nil {
		return nil, fmt.Errorf("Folder not found : %v", path)
	}
	if !o.IsDir {
		return nil, fmt.Errorf("Not a folder : %v", path)
	}
	return &fm, nil
}

func getFileManager(path string) (*FileManager, error) {
	path = gowpd.CleanPath(path)

	rs := MTP_ID.FindStringSubmatch(path)
	if len(rs) > 0 {
		id, _ := strconv.Atoi(rs[1])
		if id >= deviceCount {
			return nil, fmt.Errorf("Invaild MTP device id : %v >= %v", id, deviceCount)
		}
		return getMtpFileManager(id, path, rs[2])
	} else {
		rs = MTP_NAME.FindStringSubmatch(path)
		if len(rs) > 0 {
			id := gowpd.GetDeviceId(rs[1])
			if id < 0 {
				return nil, fmt.Errorf("Invaild MTP device name : %v", rs[1])
			}
			return getMtpFileManager(id, path, rs[2])
		} else {
			fileInfo, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("Folder not found : %v", path)
			}
			if !fileInfo.IsDir() {
				return nil, fmt.Errorf("Not a folder : %v", path)
			}
			var fm FileManager
			fm.path = path
			return &fm, nil
		}
	}
}

func listMtpFiles(d *gowpd.Device, id string, curPath string, clean bool, list map[string]*gowpd.Object) int {
	objs, _ := d.GetChildObjects(id)
	n := len(objs)
	for _, o := range objs {
		if o.Name == SYSTEM_VOLUME_INFO {
			n--
			continue
		} else if strings.HasPrefix(o.Name, ".trashed-"){
            n--
            continue
        }
		rel := filepath.Join(curPath, o.Name)
		list[rel] = o
		if o.IsDir {
			o.ChildCount = listMtpFiles(d, o.Id, rel, clean, list)
			if clean && o.ChildCount == 0 {
				delete(list, rel)
				n--
			}
		} else {
			o.ChildCount = -1
			if clean && o.Size == 0 {
				delete(list, rel)
				n--
			}
		}
	}
	return n
}

func ListMtpFiles(d *gowpd.Device, path string, clean bool) (list map[string]*gowpd.Object) {
	list = make(map[string]*gowpd.Object)
	obj := d.FindObject(path)
	if obj == nil || !obj.IsDir {
		return
	}
	listMtpFiles(d, obj.Id, "", clean, list)
	return
}

func listFiles(path string, curPath string, clean bool, list map[string]*gowpd.Object) int {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return 0
	}
	n := len(infos)
	for _, info := range infos {
		if info.Name() == SYSTEM_VOLUME_INFO {
			n--
			continue
		}
		o := gowpd.ObjectFromFileInfo(filepath.Join(path, info.Name()), info)

		rel := filepath.Join(curPath, info.Name())
		list[rel] = o
		if o.IsDir {
			o.ChildCount = listFiles(o.Id, rel, clean, list)
			if clean && o.ChildCount == 0 {
				delete(list, rel)
				n--
			}
		} else {
			o.ChildCount = -1
			if clean && o.Size == 0 {
				delete(list, rel)
				n--
			}
		}
	}
	return n
}

func (fm *FileManager) ListFiles(clean bool) map[string]*gowpd.Object {
	if fm.device != nil {
		return ListMtpFiles(fm.device, fm.path, clean)
	}
	list := make(map[string]*gowpd.Object)
	info, _ := os.Lstat(fm.path)
	if info == nil || !info.IsDir() {
		return list
	}
	listFiles(fm.path, "", clean, list)
	return list
}

func (fm *FileManager) Delete(obj *gowpd.Object) (err error) {
	if fm.device != nil {
		err = fm.device.Delete(obj.Id)
	} else {
		err = os.Remove(obj.Id)
	}
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (fm *FileManager) Mkdir(path string, obj *gowpd.Object) {
	filename := filepath.Join(fm.path, path)
	if fm.device != nil {
		parent := fm.device.FindObject(filepath.Dir(filename))
		if parent != nil {
			fm.device.CreateFolder(parent.Id, obj.Name)
		}
	} else {
		os.MkdirAll(filename, os.ModePerm)
	}
}

func SortKey(m map[string]*gowpd.Object) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func SaveList(m map[string]*gowpd.Object, path string) bool {
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer f.Close()

	keys := SortKey(m)

	w := csv.NewWriter(bufio.NewWriter(f))
	defer w.Flush()

	for _, k := range keys {
		o := m[k]
		v := []string{
			k,
			strconv.FormatInt(o.ModTime, 10),
			strconv.FormatInt(o.Size, 10),
			strconv.Itoa(o.ChildCount)}
		w.Write(v)
	}
	return true
}

func LoadList(path string) (m map[string]*gowpd.Object) {
	m = make(map[string]*gowpd.Object)
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	r := csv.NewReader(bufio.NewReader(f))
	for {
		val, err := r.Read()
		if err != nil {
			return
		}
		mtime, _ := strconv.ParseInt(val[1], 10, 64)
		size, _ := strconv.ParseInt(val[2], 10, 64)
		cnt, _ := strconv.Atoi(val[3])
		dir := (cnt >= 0)
		o := gowpd.Object{ObjectInfo: gowpd.ObjectInfo{ModTime: mtime, Size: size, IsDir: dir}, ChildCount: cnt}
		m[val[0]] = &o
	}
	return
}

func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	if err = os.Chmod(dst, srcinfo.Mode()); err != nil {
	}
	tm := srcinfo.ModTime()
	return os.Chtimes(dst, tm, tm)
}

func CopyFile(path string, obj *gowpd.Object, overwrite bool) {
	if obj.IsDir {
		dst.Mkdir(path, obj)
		return
	}
	filename := filepath.Join(dst.path, path)
	if dst.device == nil {
		if src.device == nil {
			copyFile(filepath.Join(src.path, path), filename)
		} else {
			src.device.CopyObjectFromDevice(filename, obj)
		}
	} else {
		if overwrite {
			dst.Delete(dstList[path])
		}
		parent := dst.device.FindObject(filepath.Dir(filename))
		if parent == nil {
			return
		}
		if src.device == nil {
			dst.device.CopyToDevice(parent.Id, filepath.Join(src.path, path))
		} else {
			if src.deviceId != dst.deviceId {
				reader, _ := src.device.GetReader(obj.Id)
				if reader != nil {
					dst.device.CopyObjectToDevice(parent.Id, reader, obj)
					reader.Close()
				}
			} else {
				if dst.device.CanCopy {
					dst.device.Copy(parent.Id, obj.Id)
				} else {
					reader, _ := src.device.GetReader(obj.Id)
					if reader != nil {
						var buf bytes.Buffer
						io.Copy(&buf, reader)
						reader.Close()
						dst.device.CopyObjectToDevice(parent.Id, &buf, obj)
					}
				}
			}
		}
	}
}

func checkDst() {
	dk := SortKey(dstList)
	for i := len(dk) - 1; i >= 0; i-- {
		k := dk[i]
		s, _ := srcList[k]
		if s == nil {
			fmt.Printf("- %v\n", k)
			dst.Delete(dstList[k])
		}
	}
}

type Handler func(state int, path string, obj *gowpd.Object)

func checkSrc(handle Handler) {
	sk := SortKey(srcList)
	for _, k := range sk {
		s, _ := srcList[k]
		d, _ := dstList[k]
		if d == nil {
			handle(ST_NEW, k, s)
		} else if s.ObjectInfo != d.ObjectInfo {
			if s.IsDir {
				continue
			}
			if s.ModTime >= d.ModTime {
				handle(ST_NEWER, k, s)
			} else {
				handle(ST_OLDER, k, s)
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		gowpd.Init()
		if cnt := gowpd.GetDeviceCount(); cnt > 0 {
			defer gowpd.Destroy()
			fmt.Println("------------------------------------------------------------")
			fmt.Println("MTP device\n")
			for i := 0; i < cnt; i++ {
				name := gowpd.GetDeviceName(i)
				desc := gowpd.GetDeviceDescription(i)
				if name != desc {
					name += " (" + desc + ")"
				}
				fmt.Printf("  MTP%d : %v\n", i, name)
			}
			fmt.Println("------------------------------------------------------------")
			fmt.Println()
		} else {
			gowpd.Destroy()
		}
		help()
		return
	}
	mode := "+"
	if len(os.Args) == 4 {
		switch os.Args[3] {
		case "0":
			mode = "0"
		case "-":
			mode = "-"
		case "=":
			mode = "="
		case "?":
			mode = "?"
		}
	}
	p1 := os.Args[1]
	p2 := os.Args[2]
	match := MTP_ID.MatchString(p1)
	if !match {
		match = MTP_ID.MatchString(p2)
	}
	if !match {
		match = MTP_NAME.MatchString(p1)
	}
	if !match {
		match = MTP_NAME.MatchString(p2)
	}
	if match {
		gowpd.Init()
		deviceCount = gowpd.GetDeviceCount()
		defer gowpd.Destroy()
	}

	var err error
	src, err = getFileManager(p1)
	if err != nil {
		fmt.Println(err)
		return
	} else if src.device != nil {
		defer src.device.Release()
	}

	dst, err = getFileManager(p2)
	if err != nil {
		fmt.Println(err)
		return
	} else if dst.device != nil {
		defer dst.device.Release()
	}
	if src.PathInfo == dst.PathInfo {
		fmt.Println("Error : src = dst")
		return
	}

	srcList = src.ListFiles(true)
	dstList = dst.ListFiles(true)
	excPath := filepath.Join(dst.path, LIST_FILENAME)
	switch mode {
	case "0":
		checkSrc(func(state int, k string, s *gowpd.Object) {
			switch state {
			case ST_NEW:
				fmt.Printf("[S  ] %v\n", k)
			case ST_NEWER:
				fmt.Printf("[S>D] %v\n", k)
			}
		})
		SaveList(srcList, excPath)
	case "-":
		checkDst()
	case "=":
		checkSrc(func(state int, k string, s *gowpd.Object) {
			switch state {
			case ST_NEW:
				fmt.Printf("+ %v\n", k)
				CopyFile(k, s, false)
			case ST_NEWER:
				fmt.Printf("+ %v\n", k)
				CopyFile(k, s, true)
			}
		})
		checkDst()
	case "?":
		excList := LoadList(excPath)
		checkSrc(func(state int, k string, s *gowpd.Object) {
			if k == LIST_FILENAME {
				return
			}
			switch state {
			case ST_NEW:
				fmt.Printf("[S  ]")
			case ST_NEWER:
				fmt.Printf("[S>D]")
			case ST_OLDER:
				fmt.Printf("[S<D]")
			}
			e, _ := excList[k]
			if e == nil {
				fmt.Printf(" ")
			} else {
				fmt.Printf("*")
			}
			fmt.Printf(" %v\n", k)
		})

		dk := SortKey(dstList)
		for _, k := range dk {
			s, _ := srcList[k]
			if s == nil {
				fmt.Printf("[  D]  %v\n", k)
			}
		}
	default:
		excList := LoadList(excPath)
		checkSrc(func(state int, k string, s *gowpd.Object) {
			if state == ST_NEW || state == ST_NEWER {
				e, _ := excList[k]
				if e == nil && k != LIST_FILENAME {
					fmt.Printf("+ %v\n", k)
					CopyFile(k, s, state == ST_NEWER)
				} else if s.ObjectInfo != e.ObjectInfo {
					if !s.IsDir {
						if s.ModTime >= e.ModTime && k != LIST_FILENAME {
							fmt.Printf("+ %v\n", k)
							CopyFile(k, s, state == ST_NEWER)
						}
					}
				}
			}
		})
		SaveList(srcList, excPath)
	}
}
