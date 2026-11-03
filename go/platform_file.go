package main

// platform_file.go - CTOS file system services, GetpAscb, GetUcb.
//
// File specs "{node}[vol]<dir>name" are mapped to %APPDATA%\Rats\name (node, volume
// and directory are dropped).  Error codes (CTOS erc.edf):
//   2   pErcEOM          Read/Write beyond the end of the file (Write writes nothing;
//                       Scores then grows the file with ChangeFileLength and retries)
//   203 pErcNoSuchFile   OpenFile of a missing file
//   205 pErcBadFileSpec  empty/unusable name
//   210 pErcBadFh        unknown file handle
//   218 pErcBadMode      mode other than 'mr' (6D72h) / 'mm' (6D6Dh)
//   224 pErcFileAlreadyExists  CreateFile of an existing file
//   230 pErcDiskFull     host I/O error while writing/creating
//   215 ercNoSuchVolume / 220 ercFileInUse are never returned (Scores retries forever on 220).

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
)

const (
	pErcEOM               = 2
	pErcNoSuchFile        = 203
	pErcBadFileSpec       = 205
	pErcBadFh             = 210
	pErcBadMode           = 218
	pErcFileAlreadyExists = 224
	pErcDiskFull          = 230
	pModeRead             = 0x6D72 // 'mr'
	pModeModify           = 0x6D6D // 'mm'
)

var (
	pFilesMu sync.Mutex
	pFiles          = map[uint16]*os.File{}
	pNextFh  uint16 = 1
	pDataDir string // overridable for tests / -datadir
)

func init() {
	OpenFile = pOpenFile
	CreateFile = pCreateFile
	ReadFile = pReadFile
	WriteFile = pWriteFile
	CloseFile = pCloseFile
	ChangeFileLength = pChangeFileLength
	GetFileStatus = pGetFileStatus
	GetpAscb = pGetpAscb
	GetUcb = pGetUcb
}

func pDir() string {
	if pDataDir != "" {
		return pDataDir
	}
	base := os.Getenv("APPDATA")
	if base == "" {
		base, _ = os.UserConfigDir()
	}
	return filepath.Join(base, "Rats")
}

// pMapFileSpec strips {node}, [volume] and <directory> and returns the host path.
func pMapFileSpec(spec string) (string, bool) {
	s := spec
	for _, br := range []string{"{}", "[]", "<>"} {
		if strings.HasPrefix(s, br[:1]) {
			i := strings.IndexByte(s, br[1])
			if i < 0 {
				return "", false
			}
			s = s[i+1:]
		}
	}
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, `\/:*?"<>|[]{}`) {
		return "", false
	}
	return filepath.Join(pDir(), s), true
}

func pSpec(pb Ptr, cb uint16) string { return string(Bytes(pb, int(cb))) }

func pOpenFile(pFh Ptr, pbName Ptr, cbName uint16, pbPw Ptr, cbPw uint16, mode uint16) uint16 {
	path, ok := pMapFileSpec(pSpec(pbName, cbName))
	if !ok {
		return pErcBadFileSpec
	}
	flag := os.O_RDONLY
	switch mode {
	case pModeRead:
	case pModeModify:
		flag = os.O_RDWR
	default:
		return pErcBadMode
	}
	f, err := os.OpenFile(path, flag, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return pErcNoSuchFile
		}
		return pErcBadFileSpec
	}
	pFilesMu.Lock()
	fh := pNextFh
	pNextFh++
	if pNextFh == 0 {
		pNextFh = 1
	}
	pFiles[fh] = f
	pFilesMu.Unlock()
	SetFW(pFh, fh)
	return 0
}

func pCreateFile(pbName Ptr, cbName uint16, pbPw Ptr, cbPw uint16, lfa uint32) uint16 {
	path, ok := pMapFileSpec(pSpec(pbName, cbName))
	if !ok {
		return pErcBadFileSpec
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return pErcDiskFull
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return pErcFileAlreadyExists
		}
		return pErcDiskFull
	}
	defer f.Close()
	if err := f.Truncate(int64(lfa)); err != nil {
		return pErcDiskFull
	}
	return 0
}

func pFile(fh uint16) *os.File {
	pFilesMu.Lock()
	defer pFilesMu.Unlock()
	return pFiles[fh]
}

func pReadFile(fh uint16, pBuf Ptr, cb uint16, lfa uint32, pcbRet Ptr) uint16 {
	f := pFile(fh)
	if f == nil {
		return pErcBadFh
	}
	buf := make([]byte, cb)
	n, err := f.ReadAt(buf, int64(lfa))
	PutBytes(pBuf, buf[:n])
	SetFW(pcbRet, uint16(n))
	if err == io.EOF || (err == nil && n < int(cb)) {
		return pErcEOM
	}
	if err != nil {
		return pErcEOM
	}
	return 0
}

func pWriteFile(fh uint16, pBuf Ptr, cb uint16, lfa uint32, pcbRet Ptr) uint16 {
	f := pFile(fh)
	if f == nil {
		return pErcBadFh
	}
	st, err := f.Stat()
	if err != nil {
		return pErcBadFh
	}
	if int64(lfa)+int64(cb) > st.Size() {
		SetFW(pcbRet, 0)
		return pErcEOM
	}
	n, err := f.WriteAt(Bytes(pBuf, int(cb)), int64(lfa))
	SetFW(pcbRet, uint16(n))
	if err != nil {
		return pErcDiskFull
	}
	return 0
}

func pCloseFile(fh uint16) uint16 {
	pFilesMu.Lock()
	f := pFiles[fh]
	delete(pFiles, fh)
	pFilesMu.Unlock()
	if f == nil {
		return pErcBadFh
	}
	f.Close()
	return 0
}

func pChangeFileLength(fh uint16, lfa uint32) uint16 {
	f := pFile(fh)
	if f == nil {
		return pErcBadFh
	}
	if err := f.Truncate(int64(lfa)); err != nil {
		return pErcDiskFull
	}
	return 0
}

// pGetFileStatus: code 0 = file length (DWORD); other codes return zeros.
func pGetFileStatus(fh uint16, code uint16, pRet Ptr, cbRet uint16) uint16 {
	f := pFile(fh)
	if f == nil {
		return pErcBadFh
	}
	var v [4]byte
	if code == 0 {
		st, err := f.Stat()
		if err != nil {
			return pErcBadFh
		}
		sz := uint32(st.Size())
		v = [4]byte{byte(sz), byte(sz >> 8), byte(sz >> 16), byte(sz >> 24)}
	}
	for i := uint16(0); i < cbRet; i++ {
		b := byte(0)
		if i < 4 {
			b = v[i]
		}
		SetFB(pRet.Add(i), b)
	}
	return 0
}

// pUserName: the Windows user name, at most 20 characters.
func pUserName() string {
	name := os.Getenv("USERNAME")
	if name == "" {
		if u, err := user.Current(); err == nil {
			name = u.Username
			if i := strings.LastIndexByte(name, '\\'); i >= 0 {
				name = name[i+1:]
			}
		}
	}
	if len(name) > 20 {
		name = name[:20]
	}
	return name
}

// pAscbInit fills the fake ASCB at F800:0200 (+40h length, +41h user name).
func pAscbInit() {
	for i := uint16(0); i < 0x100; i++ {
		pSysMem[pAscbOff+i] = 0
	}
	n := pUserName()
	pSysMem[pAscbOff+0x40] = byte(len(n))
	copy(pSysMem[pAscbOff+0x41:], n)
}

func pGetpAscb(ppRet Ptr) uint16 {
	SetFP(ppRet, MkPtr(SegSys, pAscbOff))
	return 0
}

// pGetUcb returns a minimal user control block: reserved WORD, defaultVol "Sys",
// defaultDir "Sys", defaultPW, prefix, defaultNode empty (sb fields).
func pGetUcb(pUcb Ptr, cb uint16) uint16 {
	var ucb [95]byte
	copy(ucb[2:], "\x03Sys")
	copy(ucb[15:], "\x03Sys")
	for i := uint16(0); i < cb && int(i) < len(ucb); i++ {
		SetFB(pUcb.Add(i), ucb[i])
	}
	return 0
}
