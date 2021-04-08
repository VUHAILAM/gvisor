// Copyright 2021 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lisafs

import (
	"math"

	"gvisor.dev/gvisor/pkg/marshal"
	"gvisor.dev/gvisor/pkg/marshal/primitive"
)

// Messages have two parts:
//  * A transport header used to decipher received messages.
//  * A byte array referred to as "payload" which contains the actual message.
//
// "dataLen" refers to the size of both combined.

// MID (message ID) is used to identify messages to parse from payload.
//
// +marshal slice:MIDSlice
type MID uint16

// These constants are used to identify their corresponding message types.
// Note that this order must be preserved across versions and new messages must
// only be appended at the end.
const (
	// Error is only used in responses to pass errors to client.
	Error MID = iota

	// Mount is used to establish connection and set up server side filesystem.
	Mount

	// Channel request starts a new channel.
	Channel

	// Fstat requests the stat(2) results for a specified file.
	Fstat

	// SetStat requests to change file attributes. Note that there is no one
	// corresponding Linux syscall. This is a conglomeration of fchmod(2),
	// fchown(2), ftruncate(2) and futimesat(2).
	SetStat

	// Walk requests to walk the specified path starting from the specified
	// directory. Server-side path traversal is terminated preemptively on
	// symlinks and `..` entries because they can cause non-linear traversal.
	Walk

	// OpenAt is loosely analogous to openat(2). It does not perform any walk. It
	// merely duplicates the specified FD with open flags passed.
	OpenAt

	// OpenCreateAt is loosely analogous to openat(2) with O_CREAT|O_EXCL added
	// to flags. It also returns the newly created file inode.
	OpenCreateAt

	// Close is analogous to close(2) but can work on multiple FDs.
	Close

	// Fsync requests to fsync(2) the specified FDs.
	Fsync

	// PWrite requests to pwrite(2) to the specified FD.
	PWrite

	// PRead requests to pread(2) from the specified FD.
	PRead

	// MkdirAt is analogous to mkdirat(2).
	MkdirAt

	// MknodAt is analogous to mknodat(2).
	MknodAt

	// SymlinkAt is analogous to symlinkat(2).
	SymlinkAt

	// LinkAt is analogous to linkat(2).
	LinkAt

	// FStatFS is analogous to fstatfs(2).
	FStatFS

	// FAllocate is analogous to fallocate(2).
	FAllocate

	// ReadLinkAt is analogous to readlinkat(2).
	ReadLinkAt

	// FFlust is loosely analogous to fflush(3). Its behavior is implementation
	// dependent and might not even be supported in server implementations.
	FFlush

	// Connect is loosely analogous to connect(2).
	Connect

	// UnlinkAt is analogous to unlinkat(2).
	UnlinkAt

	// RenameAt is loosely analogous to renameat(2).
	RenameAt

	// Getdents64 is analogous to getdents64(2).
	Getdents64

	// FGetXattr is analogous to fgetxattr(2).
	FGetXattr

	// FSetXattr is analogous to fsetxattr(2).
	FSetXattr

	// FListXattr is analogous to flistxattr(2).
	FListXattr

	// FRemoveXattr is analogous to fremovexattr(2).
	FRemoveXattr
)

const (
	// MaxMessageSize is the largest possible message in bytes.
	MaxMessageSize uint32 = 1 << 20

	// NoUID is a sentinel used to indicate no valid UID.
	NoUID UID = math.MaxUint32

	// NoGID is a sentinel used to indicate no valid GID.
	NoGID GID = math.MaxUint32
)

// UID represents a user ID.
//
// +marshal
type UID uint32

// Ok returns true if uid is not NoUID.
func (uid UID) Ok() bool {
	return uid != NoUID
}

// GID represents a group ID.
//
// +marshal
type GID uint32

// Ok returns true if gid is not NoGID.
func (gid GID) Ok() bool {
	return gid != NoGID
}

// sockHeader is the header present in front of each message received on a UDS.
//
// +marshal
type sockHeader struct {
	size    uint32
	message MID
	_       uint16
}

// channelHeader is the header present in front of each message received on
// flipcall endpoint.
//
// +marshal
type channelHeader struct {
	message MID
	numFDs  uint8
	_       uint8
}

// SizedString represents a string in memory. The string bytes are preceded by
// a uint32 signifying the string length.
//
// +marshal dynamic
type SizedString string

var _ marshal.Marshallable = (*SizedString)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (s *SizedString) SizeBytes() int {
	return (*primitive.Uint32)(nil).SizeBytes() + len(*s)
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (s *SizedString) MarshalBytes(dst []byte) {
	strLen := primitive.Uint32(len(*s))
	strLen.MarshalBytes(dst)
	dst = dst[strLen.SizeBytes():]
	// Copy without any allocation.
	copy(dst[:strLen], *s)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (s *SizedString) UnmarshalBytes(src []byte) {
	var strLen primitive.Uint32
	strLen.UnmarshalBytes(src)
	src = src[strLen.SizeBytes():]
	// Take the hit, this leads to an allocation + memcpy. No way around it.
	*s = SizedString(src[:strLen])
}

// StringArray represents an array of SizedStrings in memory. The array data is
// preceded by a uint32 signifying the array length.
//
// +marshal dynamic
type StringArray []string

var _ marshal.Marshallable = (*StringArray)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (s *StringArray) SizeBytes() int {
	size := (*primitive.Uint32)(nil).SizeBytes()
	for _, str := range *s {
		sstr := SizedString(str)
		size += sstr.SizeBytes()
	}
	return size
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (s *StringArray) MarshalBytes(dst []byte) {
	arrLen := primitive.Uint32(len(*s))
	arrLen.MarshalBytes(dst)
	dst = dst[arrLen.SizeBytes():]
	for _, str := range *s {
		sstr := SizedString(str)
		sstr.MarshalBytes(dst)
		dst = dst[sstr.SizeBytes():]
	}
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (s *StringArray) UnmarshalBytes(src []byte) {
	var arrLen primitive.Uint32
	arrLen.UnmarshalBytes(src)
	src = src[arrLen.SizeBytes():]

	if cap(*s) < int(arrLen) {
		*s = make([]string, arrLen)
	} else {
		*s = (*s)[:arrLen]
	}

	for i := primitive.Uint32(0); i < arrLen; i++ {
		var sstr SizedString
		sstr.UnmarshalBytes(src)
		src = src[sstr.SizeBytes():]
		(*s)[i] = string(sstr)
	}
}

// Inode represents an inode on the remote filesystem.
//
// +marshal slice:InodeSlice
type Inode struct {
	ControlFD FDID
	_         uint32
	Stat      StatX
}

// MountReq represents a Mount request.
//
// +marshal dynamic
type MountReq struct {
	MountPath SizedString
}

var _ marshal.Marshallable = (*MountReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (m *MountReq) SizeBytes() int {
	return m.MountPath.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (m *MountReq) MarshalBytes(dst []byte) {
	m.MountPath.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (m *MountReq) UnmarshalBytes(src []byte) {
	m.MountPath.UnmarshalBytes(src)
}

// MountResp represents a Mount response.
//
// +marshal dynamic
type MountResp struct {
	Root          Inode
	MaxM          MID
	UnsupportedMs []MID
}

var _ marshal.Marshallable = (*MountResp)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (m *MountResp) SizeBytes() int {
	return m.Root.SizeBytes() +
		m.MaxM.SizeBytes() +
		(*primitive.Uint16)(nil).SizeBytes() +
		(len(m.UnsupportedMs) * (*MID)(nil).SizeBytes())
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (m *MountResp) MarshalBytes(dst []byte) {
	m.Root.MarshalBytes(dst)
	dst = dst[m.Root.SizeBytes():]
	m.MaxM.MarshalBytes(dst)
	dst = dst[m.MaxM.SizeBytes():]
	numUnsupported := primitive.Uint16(len(m.UnsupportedMs))
	numUnsupported.MarshalBytes(dst)
	dst = dst[numUnsupported.SizeBytes():]
	MarshalUnsafeMIDSlice(m.UnsupportedMs, dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (m *MountResp) UnmarshalBytes(src []byte) {
	m.Root.UnmarshalBytes(src)
	src = src[m.Root.SizeBytes():]
	m.MaxM.UnmarshalBytes(src)
	src = src[m.MaxM.SizeBytes():]
	var numUnsupported primitive.Uint16
	numUnsupported.UnmarshalBytes(src)
	src = src[numUnsupported.SizeBytes():]
	m.UnsupportedMs = make([]MID, numUnsupported)
	UnmarshalUnsafeMIDSlice(m.UnsupportedMs, src)
}

// ChannelResp is the response to the create channel request.
//
// +marshal
type ChannelResp struct {
	dataOffset int64
	dataLength uint64
}

// ErrorRes is returned to represent an error while handling a request.
// A field holding value 0 indicates no error on that field.
//
// +marshal
type ErrorRes struct {
	errno uint32
}

// Timespec is similar to `struct timespec` in Linux.
//
// +marshal
type Timespec struct {
	Sec  int64
	Nsec int64
}

// StatX is used to communicate stat(2) results.
//
// +marshal
type StatX struct {
	Mask    uint32
	Mode    uint32
	Nlink   uint32
	Blksize uint32
	Dev     uint64
	Ino     uint64
	UID     UID
	GID     GID
	Rdev    uint64
	Size    uint64
	Blocks  uint64
	Atime   Timespec
	Mtime   Timespec
	Ctime   Timespec
	Btime   Timespec
}

// StatReq requests the stat results for the specified FD.
//
// +marshal
type StatReq struct {
	FD FDID
}

// SetStatReq is used to set attributeds on FDs.
//
// +marshal
type SetStatReq struct {
	FD    FDID
	_     uint32
	Mask  uint32
	Mode  uint32 // Only permissions part is settable.
	UID   UID
	GID   GID
	Size  uint64
	Atime Timespec
	Mtime Timespec
}

// SetStatResp is used to communicate SetStat results. It contains a mask
// representing the failed changes.
//
// +marshal
type SetStatResp struct {
	FailureMask uint32
}

// WalkReq is used to request to walk multiple path components at once.
//
// +marshal dynamic
type WalkReq struct {
	DirFD FDID
	Path  StringArray
}

var _ marshal.Marshallable = (*WalkReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (w *WalkReq) SizeBytes() int {
	return w.DirFD.SizeBytes() + w.Path.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (w *WalkReq) MarshalBytes(dst []byte) {
	w.DirFD.MarshalBytes(dst)
	dst = dst[w.DirFD.SizeBytes():]
	w.Path.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (w *WalkReq) UnmarshalBytes(src []byte) {
	w.DirFD.UnmarshalBytes(src)
	src = src[w.DirFD.SizeBytes():]
	w.Path.UnmarshalBytes(src)
}

// WalkResp is used to communicate the inodes walked by the server.
//
// +marshal dynamic
type WalkResp struct {
	Inodes []Inode
}

var _ marshal.Marshallable = (*WalkResp)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (w *WalkResp) SizeBytes() int {
	return (*primitive.Uint32)(nil).SizeBytes() + (len(w.Inodes) * (*Inode)(nil).SizeBytes())
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (w *WalkResp) MarshalBytes(dst []byte) {
	numInodes := primitive.Uint32(len(w.Inodes))
	numInodes.MarshalBytes(dst)
	dst = dst[numInodes.SizeBytes():]

	MarshalUnsafeInodeSlice(w.Inodes, dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (w *WalkResp) UnmarshalBytes(src []byte) {
	var numInodes primitive.Uint32
	numInodes.UnmarshalBytes(src)
	src = src[numInodes.SizeBytes():]

	if cap(w.Inodes) < int(numInodes) {
		w.Inodes = make([]Inode, numInodes)
	} else {
		w.Inodes = w.Inodes[:numInodes]
	}
	UnmarshalUnsafeInodeSlice(w.Inodes, src)
}

// OpenAtReq is used to open existing FDs with the specified flags.
//
// +marshal
type OpenAtReq struct {
	FD    FDID
	Flags uint32
}

// OpenAtResp is used to communicate the newly created FD.
//
// +marshal
type OpenAtResp struct {
	NewFD FDID
}

// OpenCreateAtReq is used to make OpenCreateAt requests.
//
// +marshal dynamic
type OpenCreateAtReq struct {
	DirFD FDID
	Name  SizedString
	Flags primitive.Uint32
	Mode  primitive.Uint32
	UID   UID
	GID   GID
}

var _ marshal.Marshallable = (*OpenCreateAtReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (o *OpenCreateAtReq) SizeBytes() int {
	return o.DirFD.SizeBytes() + o.Name.SizeBytes() + o.Flags.SizeBytes() + o.Mode.SizeBytes() + o.UID.SizeBytes() + o.GID.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (o *OpenCreateAtReq) MarshalBytes(dst []byte) {
	o.DirFD.MarshalBytes(dst)
	dst = dst[o.DirFD.SizeBytes():]
	o.Name.MarshalBytes(dst)
	dst = dst[o.Name.SizeBytes():]
	o.Flags.MarshalBytes(dst)
	dst = dst[o.Flags.SizeBytes():]
	o.Mode.MarshalBytes(dst)
	dst = dst[o.Mode.SizeBytes():]
	o.UID.MarshalBytes(dst)
	dst = dst[o.UID.SizeBytes():]
	o.GID.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (o *OpenCreateAtReq) UnmarshalBytes(src []byte) {
	o.DirFD.UnmarshalBytes(src)
	src = src[o.DirFD.SizeBytes():]
	o.Name.UnmarshalBytes(src)
	src = src[o.Name.SizeBytes():]
	o.Flags.UnmarshalBytes(src)
	src = src[o.Flags.SizeBytes():]
	o.Mode.UnmarshalBytes(src)
	src = src[o.Mode.SizeBytes():]
	o.UID.UnmarshalBytes(src)
	src = src[o.UID.SizeBytes():]
	o.GID.UnmarshalBytes(src)
}

// OpenCreateAtResp is used to communicate successful OpenCreateAt results.
//
// +marshal
type OpenCreateAtResp struct {
	Child Inode
	NewFD FDID
	_     uint32
}

// FdArray is a utility struct which implements a marshallable type for
// communicating an array of FDIDs. In memory, the array data is preceded by a
// uint32 denoting the array length.
//
// +marshal dynamic
type FdArray []FDID

var _ marshal.Marshallable = (*FdArray)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (f *FdArray) SizeBytes() int {
	return (*primitive.Uint32)(nil).SizeBytes() + (len(*f) * (*FDID)(nil).SizeBytes())
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (f *FdArray) MarshalBytes(dst []byte) {
	arrLen := primitive.Uint32(len(*f))
	arrLen.MarshalBytes(dst)
	dst = dst[arrLen.SizeBytes():]
	MarshalUnsafeFDIDSlice(*f, dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (f *FdArray) UnmarshalBytes(src []byte) {
	var arrLen primitive.Uint32
	arrLen.UnmarshalBytes(src)
	src = src[arrLen.SizeBytes():]
	if cap(*f) < int(arrLen) {
		*f = make(FdArray, arrLen)
	} else {
		*f = (*f)[:arrLen]
	}
	UnmarshalUnsafeFDIDSlice(*f, src)
}

// CloseReq is used to close(2) FDs.
//
// +marshal dynamic
type CloseReq struct {
	FDs FdArray
}

var _ marshal.Marshallable = (*CloseReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (c *CloseReq) SizeBytes() int {
	return c.FDs.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (c *CloseReq) MarshalBytes(dst []byte) {
	c.FDs.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (c *CloseReq) UnmarshalBytes(src []byte) {
	c.FDs.UnmarshalBytes(src)
}

// FsyncReq is used to fsync(2) FDs.
//
// +marshal dynamic
type FsyncReq struct {
	FDs FdArray
}

var _ marshal.Marshallable = (*FsyncReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (f *FsyncReq) SizeBytes() int {
	return f.FDs.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (f *FsyncReq) MarshalBytes(dst []byte) {
	f.FDs.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (f *FsyncReq) UnmarshalBytes(src []byte) {
	f.FDs.UnmarshalBytes(src)
}

// PReadReq is used to pread(2) on an FD.
//
// +marshal
type PReadReq struct {
	Offset uint64
	FD     FDID
	Count  uint32
}

// PReadResp is used to return the result of pread(2).
//
// +marshal dynamic
type PReadResp struct {
	NumBytes primitive.Uint32
	Buf      []byte
}

var _ marshal.Marshallable = (*PReadResp)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (r *PReadResp) SizeBytes() int {
	return r.NumBytes.SizeBytes() + int(r.NumBytes)
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (r *PReadResp) MarshalBytes(dst []byte) {
	r.NumBytes.MarshalBytes(dst)
	dst = dst[r.NumBytes.SizeBytes():]
	copy(dst[:r.NumBytes], r.Buf[:r.NumBytes])
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (r *PReadResp) UnmarshalBytes(src []byte) {
	r.NumBytes.UnmarshalBytes(src)
	src = src[r.NumBytes.SizeBytes():]

	// We expect the client to have already allocated r.Buf. r.Buf probably
	// (optimally) points to usermem. Directly copy into that.
	copy(r.Buf[:r.NumBytes], src[:r.NumBytes])
}

// PWriteReq is used to pwrite(2) on an FD.
//
// +marshal dynamic
type PWriteReq struct {
	Offset   primitive.Uint64
	FD       FDID
	NumBytes primitive.Uint32
	Buf      []byte
}

var _ marshal.Marshallable = (*PWriteReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (w *PWriteReq) SizeBytes() int {
	return w.Offset.SizeBytes() + w.FD.SizeBytes() + w.NumBytes.SizeBytes() + int(w.NumBytes)
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (w *PWriteReq) MarshalBytes(dst []byte) {
	w.Offset.MarshalBytes(dst)
	dst = dst[w.Offset.SizeBytes():]
	w.FD.MarshalBytes(dst)
	dst = dst[w.FD.SizeBytes():]
	w.NumBytes.MarshalBytes(dst)
	dst = dst[w.NumBytes.SizeBytes():]
	copy(dst[:w.NumBytes], w.Buf[:w.NumBytes])
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (w *PWriteReq) UnmarshalBytes(src []byte) {
	w.Offset.UnmarshalBytes(src)
	src = src[w.Offset.SizeBytes():]
	w.FD.UnmarshalBytes(src)
	src = src[w.FD.SizeBytes():]
	w.NumBytes.UnmarshalBytes(src)
	src = src[w.NumBytes.SizeBytes():]

	// This is an optimization. Assuming that the server is making this call, it
	// is safe to just point to src rather than allocating and copying.
	w.Buf = src[:w.NumBytes]
}

// PWriteResp is used to return the result of pwrite(2).
//
// +marshal
type PWriteResp struct {
	Count uint64
}

// +marshal dynamic
type mkCommon struct {
	DirFD FDID
	Name  SizedString
	Mode  primitive.Uint32
	UID   UID
	GID   GID
}

var _ marshal.Marshallable = (*mkCommon)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (m *mkCommon) SizeBytes() int {
	return m.DirFD.SizeBytes() + m.Name.SizeBytes() + m.Mode.SizeBytes() + m.UID.SizeBytes() + m.GID.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (m *mkCommon) MarshalBytes(dst []byte) {
	m.DirFD.MarshalBytes(dst)
	dst = dst[m.DirFD.SizeBytes():]
	m.Name.MarshalBytes(dst)
	dst = dst[m.Name.SizeBytes():]
	m.Mode.MarshalBytes(dst)
	dst = dst[m.Mode.SizeBytes():]
	m.UID.MarshalBytes(dst)
	dst = dst[m.UID.SizeBytes():]
	m.GID.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (m *mkCommon) UnmarshalBytes(src []byte) {
	m.DirFD.UnmarshalBytes(src)
	src = src[m.DirFD.SizeBytes():]
	m.Name.UnmarshalBytes(src)
	src = src[m.Name.SizeBytes():]
	m.Mode.UnmarshalBytes(src)
	src = src[m.Mode.SizeBytes():]
	m.UID.UnmarshalBytes(src)
	src = src[m.UID.SizeBytes():]
	m.GID.UnmarshalBytes(src)
}

// MkdirAtReq is used to make MkdirAt requests.
type MkdirAtReq struct {
	mkCommon
}

// MkdirAtResp is the response to a successful MkdirAt request.
//
// +marshal
type MkdirAtResp struct {
	ChildDir Inode
}

// MknodAtReq is used to make MknodAt requests.
//
// +marshal dynamic
type MknodAtReq struct {
	mkCommon
	Minor primitive.Uint32
	Major primitive.Uint32
}

var _ marshal.Marshallable = (*MknodAtReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (m *MknodAtReq) SizeBytes() int {
	return m.mkCommon.SizeBytes() + m.Minor.SizeBytes() + m.Major.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (m *MknodAtReq) MarshalBytes(dst []byte) {
	m.mkCommon.MarshalBytes(dst)
	dst = dst[m.mkCommon.SizeBytes():]
	m.Minor.MarshalBytes(dst)
	dst = dst[m.Minor.SizeBytes():]
	m.Major.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (m *MknodAtReq) UnmarshalBytes(src []byte) {
	m.mkCommon.UnmarshalBytes(src)
	src = src[m.mkCommon.SizeBytes():]
	m.Minor.UnmarshalBytes(src)
	src = src[m.Minor.SizeBytes():]
	m.Major.UnmarshalBytes(src)
}

// MknodAtResp is the response to a successful MknodAt request.
//
// +marshal
type MknodAtResp struct {
	Child Inode
}

// SymlinkAtReq is used to make SymlinkAt request.
//
// +marshal dynamic
type SymlinkAtReq struct {
	DirFD  FDID
	Name   SizedString
	Target SizedString
	UID    UID
	GID    GID
}

var _ marshal.Marshallable = (*SymlinkAtReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (s *SymlinkAtReq) SizeBytes() int {
	return s.DirFD.SizeBytes() + s.Name.SizeBytes() + s.Target.SizeBytes() + s.UID.SizeBytes() + s.GID.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (s *SymlinkAtReq) MarshalBytes(dst []byte) {
	s.DirFD.MarshalBytes(dst)
	dst = dst[s.DirFD.SizeBytes():]
	s.Name.MarshalBytes(dst)
	dst = dst[s.Name.SizeBytes():]
	s.Target.MarshalBytes(dst)
	dst = dst[s.Target.SizeBytes():]
	s.UID.MarshalBytes(dst)
	dst = dst[s.UID.SizeBytes():]
	s.GID.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (s *SymlinkAtReq) UnmarshalBytes(src []byte) {
	s.DirFD.UnmarshalBytes(src)
	src = src[s.DirFD.SizeBytes():]
	s.Name.UnmarshalBytes(src)
	src = src[s.Name.SizeBytes():]
	s.Target.UnmarshalBytes(src)
	src = src[s.Target.SizeBytes():]
	s.UID.UnmarshalBytes(src)
	src = src[s.UID.SizeBytes():]
	s.GID.UnmarshalBytes(src)
}

// SymlinkAtResp is the response to a successful SymlinkAt request.
//
// +marshal
type SymlinkAtResp struct {
	Symlink Inode
}

// LinkAtReq is used to make LinkAt requests.
//
// +marshal dynamic
type LinkAtReq struct {
	DirFD  FDID
	Target FDID
	Name   SizedString
}

var _ marshal.Marshallable = (*LinkAtReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (l *LinkAtReq) SizeBytes() int {
	return l.DirFD.SizeBytes() + l.Target.SizeBytes() + l.Name.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (l *LinkAtReq) MarshalBytes(dst []byte) {
	l.DirFD.MarshalBytes(dst)
	dst = dst[l.DirFD.SizeBytes():]
	l.Target.MarshalBytes(dst)
	dst = dst[l.Target.SizeBytes():]
	l.Name.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (l *LinkAtReq) UnmarshalBytes(src []byte) {
	l.DirFD.UnmarshalBytes(src)
	src = src[l.DirFD.SizeBytes():]
	l.Target.UnmarshalBytes(src)
	src = src[l.Target.SizeBytes():]
	l.Name.UnmarshalBytes(src)
}

// LinkAtResp is used to respond to a successful LinkAt request.
//
// +marshal
type LinkAtResp struct {
	Link Inode
}

// FStatFSReq is used to request StatFS results for the specified FD.
//
// +marshal
type FStatFSReq struct {
	FD FDID
}

// FStatFSResp is responded to a successful FStatFS request.
//
// +marshal
type FStatFSResp struct {
	Type            uint64
	BlockSize       int64
	Blocks          uint64
	BlocksFree      uint64
	BlocksAvailable uint64
	Files           uint64
	FilesFree       uint64
	NameLength      uint64
}

// FAllocateReq is used to request to fallocate(2) an FD. This has no response.
//
// +marshal
type FAllocateReq struct {
	FD     FDID
	Mode   uint32
	Offset uint64
	Length uint64
}

// ReadLinkAtReq is used to readlinkat(2) at the specified FD.
//
// +marshal
type ReadLinkAtReq struct {
	FD FDID
}

// ReadLinkAtResp is used to communicate ReadLinkAt results.
//
// +marshal dynamic
type ReadLinkAtResp struct {
	Target SizedString
}

var _ marshal.Marshallable = (*ReadLinkAtResp)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (r *ReadLinkAtResp) SizeBytes() int {
	return r.Target.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (r *ReadLinkAtResp) MarshalBytes(dst []byte) {
	r.Target.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (r *ReadLinkAtResp) UnmarshalBytes(src []byte) {
	r.Target.UnmarshalBytes(src)
}

// FFlushReq is used to make FFlush requests.
//
// +marshal
type FFlushReq struct {
	FD FDID
}

// ConnectReq is used to make a Connect request.
//
// +marshal
type ConnectReq struct {
	FD FDID
	// SockType is used to specify the socket type to connect to. As a special
	// case, SockType = 0 means that the socket type does not matter and the
	// requester will accept any socket type.
	SockType uint32
}

// UnlinkAtReq is used to make UnlinkAt request.
//
// +marshal dynamic
type UnlinkAtReq struct {
	DirFD FDID
	Name  SizedString
	Flags primitive.Uint32
}

var _ marshal.Marshallable = (*UnlinkAtReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (u *UnlinkAtReq) SizeBytes() int {
	return u.DirFD.SizeBytes() + u.Name.SizeBytes() + u.Flags.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (u *UnlinkAtReq) MarshalBytes(dst []byte) {
	u.DirFD.MarshalBytes(dst)
	dst = dst[u.DirFD.SizeBytes():]
	u.Name.MarshalBytes(dst)
	dst = dst[u.Name.SizeBytes():]
	u.Flags.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (u *UnlinkAtReq) UnmarshalBytes(src []byte) {
	u.DirFD.UnmarshalBytes(src)
	src = src[u.DirFD.SizeBytes():]
	u.Name.UnmarshalBytes(src)
	src = src[u.Name.SizeBytes():]
	u.Flags.UnmarshalBytes(src)
}

// RenameAtReq is used to make Rename requests. Note that the request takes in
// the to-be-renamed file's FD instead of oldDir and oldName like renameat(2).
//
// +marshal dynamic
type RenameAtReq struct {
	Renamed FDID
	NewDir  FDID
	NewName SizedString
}

var _ marshal.Marshallable = (*RenameAtReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (r *RenameAtReq) SizeBytes() int {
	return r.Renamed.SizeBytes() + r.NewDir.SizeBytes() + r.NewName.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (r *RenameAtReq) MarshalBytes(dst []byte) {
	r.Renamed.MarshalBytes(dst)
	dst = dst[r.Renamed.SizeBytes():]
	r.NewDir.MarshalBytes(dst)
	dst = dst[r.NewDir.SizeBytes():]
	r.NewName.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (r *RenameAtReq) UnmarshalBytes(src []byte) {
	r.Renamed.UnmarshalBytes(src)
	src = src[r.Renamed.SizeBytes():]
	r.NewDir.UnmarshalBytes(src)
	src = src[r.NewDir.SizeBytes():]
	r.NewName.UnmarshalBytes(src)
}

// Getdents64Req is used to make Getdents64 requests.
//
// +marshal
type Getdents64Req struct {
	DirFD FDID
	// Count is the number of bytes to read. A negative value of Count is used to
	// indicate that the implementation must lseek(0, SEEK_SET) before calling
	// getdents64(2). Implementations must use the absolute value of Count to
	// determine the number of bytes to read.
	Count int32
}

// Dirent64 is analogous to struct linux_dirent64.
//
// +marshal dynamic
type Dirent64 struct {
	Ino  primitive.Uint64
	Dev  primitive.Uint64
	Off  primitive.Uint64
	Type primitive.Uint8
	Name SizedString
}

var _ marshal.Marshallable = (*Dirent64)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (d *Dirent64) SizeBytes() int {
	return d.Ino.SizeBytes() + d.Dev.SizeBytes() + d.Off.SizeBytes() + d.Type.SizeBytes() + d.Name.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (d *Dirent64) MarshalBytes(dst []byte) {
	d.Ino.MarshalBytes(dst)
	dst = dst[d.Ino.SizeBytes():]
	d.Dev.MarshalBytes(dst)
	dst = dst[d.Dev.SizeBytes():]
	d.Off.MarshalBytes(dst)
	dst = dst[d.Off.SizeBytes():]
	d.Type.MarshalBytes(dst)
	dst = dst[d.Type.SizeBytes():]
	d.Name.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (d *Dirent64) UnmarshalBytes(src []byte) {
	d.Ino.UnmarshalBytes(src)
	src = src[d.Ino.SizeBytes():]
	d.Dev.UnmarshalBytes(src)
	src = src[d.Dev.SizeBytes():]
	d.Off.UnmarshalBytes(src)
	src = src[d.Off.SizeBytes():]
	d.Type.UnmarshalBytes(src)
	src = src[d.Type.SizeBytes():]
	d.Name.UnmarshalBytes(src)
}

// Getdents64Resp is used to communicate getdents64 results.
//
// +marshal dynamic
type Getdents64Resp struct {
	Dirents []Dirent64
}

var _ marshal.Marshallable = (*Getdents64Resp)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (g *Getdents64Resp) SizeBytes() int {
	ret := (*primitive.Uint32)(nil).SizeBytes()
	for i := range g.Dirents {
		ret += g.Dirents[i].SizeBytes()
	}
	return ret
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (g *Getdents64Resp) MarshalBytes(dst []byte) {
	numDirents := primitive.Uint32(len(g.Dirents))
	numDirents.MarshalBytes(dst)
	dst = dst[numDirents.SizeBytes():]
	for i := range g.Dirents {
		g.Dirents[i].MarshalBytes(dst)
		dst = dst[g.Dirents[i].SizeBytes():]
	}
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (g *Getdents64Resp) UnmarshalBytes(src []byte) {
	var numDirents primitive.Uint32
	numDirents.UnmarshalBytes(src)
	g.Dirents = make([]Dirent64, numDirents)
	src = src[numDirents.SizeBytes():]
	for i := range g.Dirents {
		g.Dirents[i].UnmarshalBytes(src)
		src = src[g.Dirents[i].SizeBytes():]
	}
}

// FGetXattrReq is used to make FGetXattr requests. The response to this is
// just a SizedString containing the xattr value.
//
// +marshal dynamic
type FGetXattrReq struct {
	FD      FDID
	BufSize primitive.Uint32
	Name    SizedString
}

var _ marshal.Marshallable = (*FGetXattrReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (g *FGetXattrReq) SizeBytes() int {
	return g.FD.SizeBytes() + g.BufSize.SizeBytes() + g.Name.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (g *FGetXattrReq) MarshalBytes(dst []byte) {
	g.FD.MarshalBytes(dst)
	dst = dst[g.FD.SizeBytes():]
	g.BufSize.MarshalBytes(dst)
	dst = dst[g.BufSize.SizeBytes():]
	g.Name.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (g *FGetXattrReq) UnmarshalBytes(src []byte) {
	g.FD.UnmarshalBytes(src)
	src = src[g.FD.SizeBytes():]
	g.BufSize.UnmarshalBytes(src)
	src = src[g.BufSize.SizeBytes():]
	g.Name.UnmarshalBytes(src)
}

// FGetXattrResp is used to respond to FGetXattr request.
//
// +marshal dynamic
type FGetXattrResp struct {
	Value SizedString
}

var _ marshal.Marshallable = (*FGetXattrResp)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (g *FGetXattrResp) SizeBytes() int {
	return g.Value.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (g *FGetXattrResp) MarshalBytes(dst []byte) {
	g.Value.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (g *FGetXattrResp) UnmarshalBytes(src []byte) {
	g.Value.UnmarshalBytes(src)
}

// FSetXattrReq is used to make FSetXattr requests. It has no response.
//
// +marshal dynamic
type FSetXattrReq struct {
	FD    FDID
	Flags primitive.Uint32
	Name  SizedString
	Value SizedString
}

var _ marshal.Marshallable = (*FSetXattrReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (s *FSetXattrReq) SizeBytes() int {
	return s.FD.SizeBytes() + s.Flags.SizeBytes() + s.Name.SizeBytes() + s.Value.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (s *FSetXattrReq) MarshalBytes(dst []byte) {
	s.FD.MarshalBytes(dst)
	dst = dst[s.FD.SizeBytes():]
	s.Flags.MarshalBytes(dst)
	dst = dst[s.Flags.SizeBytes():]
	s.Name.MarshalBytes(dst)
	dst = dst[s.Name.SizeBytes():]
	s.Value.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (s *FSetXattrReq) UnmarshalBytes(src []byte) {
	s.FD.UnmarshalBytes(src)
	src = src[s.FD.SizeBytes():]
	s.Flags.UnmarshalBytes(src)
	src = src[s.Flags.SizeBytes():]
	s.Name.UnmarshalBytes(src)
	src = src[s.Name.SizeBytes():]
	s.Value.UnmarshalBytes(src)
}

// FRemoveXattrReq is used to make FRemoveXattr requests. It has no response.
//
// +marshal dynamic
type FRemoveXattrReq struct {
	FD   FDID
	Name SizedString
}

var _ marshal.Marshallable = (*FRemoveXattrReq)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (r *FRemoveXattrReq) SizeBytes() int {
	return r.FD.SizeBytes() + r.Name.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (r *FRemoveXattrReq) MarshalBytes(dst []byte) {
	r.FD.MarshalBytes(dst)
	dst = dst[r.FD.SizeBytes():]
	r.Name.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (r *FRemoveXattrReq) UnmarshalBytes(src []byte) {
	r.FD.UnmarshalBytes(src)
	src = src[r.FD.SizeBytes():]
	r.Name.UnmarshalBytes(src)
}

// FListXattrReq is used to make FListXattr requests.
//
// +marshal
type FListXattrReq struct {
	FD   FDID
	_    uint32
	Size uint64
}

// FListXattrResp is used to respond to FListXattr requests.
//
// +marshal dynamic
type FListXattrResp struct {
	Xattrs StringArray
}

var _ marshal.Marshallable = (*FListXattrResp)(nil)

// SizeBytes implements marshal.Marshallable.SizeBytes.
func (l *FListXattrResp) SizeBytes() int {
	return l.Xattrs.SizeBytes()
}

// MarshalBytes implements marshal.Marshallable.MarshalBytes.
func (l *FListXattrResp) MarshalBytes(dst []byte) {
	l.Xattrs.MarshalBytes(dst)
}

// UnmarshalBytes implements marshal.Marshallable.UnmarshalBytes.
func (l *FListXattrResp) UnmarshalBytes(src []byte) {
	l.Xattrs.UnmarshalBytes(src)
}
