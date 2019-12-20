# kfilesync

 Sync MTP devices(Android devices, USB memory, etc) to Windows PC.
 Sync between MTP devices is also possible.

## Usage
```
kfilesync src dst [mode]

  src   Source folder
            ex) MTP0:\DCIM  - DCIM folder of MTP device with id = 0
  dst   Destination folder
  mode  Sync mode
        +  Copy new files from source folder (default)
        0  Init sync. Copy no files and just make list of files in source folder.
           Current files in source folder will not be copied in next sync.
        -  Delete files which is not in source folder.
        =  Make destination folder equal to source folder.
        ?  Show differences only.
```