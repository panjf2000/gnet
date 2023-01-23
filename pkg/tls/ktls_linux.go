//go:build linux
// +build linux

package tls

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	TCP_ULP = 31
	SOL_TLS = 282

	TLS_TX               = 1
	TLS_RX               = 2
	TLS_TX_ZEROCOPY_RO   = 3 // TX zerocopy (only sendfile now)
	TLS_RX_EXPECT_NO_PAD = 4 // Attempt opportunistic zero-copy, TLS 1.3 only

	TLS_SET_RECORD_TYPE = 1
	TLS_GET_RECORD_TYPE = 2

	kTLSOverhead = 16
)

var (
	kTLSSupport bool

	// kTLSSupportTX is true when kTLSSupport is true
	kTLSSupportTX bool
	kTLSSupportRX bool

	// kTLSSupportAESGCM128 is true when kTLSSupport is true
	kTLSSupportAESGCM128        bool
	kTLSSupportAESGCM256        bool
	kTLSSupportCHACHA20POLY1305 bool

	kTLSSupportTLS13 bool
)

func init() {
	// when kernel tls module enabled, /sys/module/tls is available
	_, err := os.Stat("/sys/module/tls")
	if err != nil {
		Debugln("kTLS: kernel tls module not enabled")
		return
	}
	kTLSSupport = true && kTLSEnabled
	Debugf("kTLS Enabled Status: %v\n", kTLSSupport)

	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		Debugf("kTLS: call uname failed %v", err)
		return
	}

	var buf [65]byte
	for i, b := range uname.Release {
		buf[i] = byte(b)
	}
	release := string(buf[:])
	if i := strings.Index(release, "\x00"); i != -1 {
		release = release[:i]
	}
	majorRelease := release[:strings.Index(release, ".")]
	minorRelease := strings.TrimLeft(release, majorRelease+".")
	minorRelease = minorRelease[:strings.Index(minorRelease, ".")]
	major, err := strconv.Atoi(majorRelease)
	if err != nil {
		Debugf("kTLS: parse major release failed %v", err)
		return
	}
	minor, err := strconv.Atoi(minorRelease)
	if err != nil {
		Debugf("kTLS: parse minor release failed %v", err)
		return
	}

	if (major == 4 && minor >= 13) || major > 4 {
		kTLSSupportTX = true
		kTLSSupportAESGCM128 = true
	}

	if (major == 4 && minor >= 17) || major > 4 {
		kTLSSupportRX = true
	}

	if (major == 5 && minor >= 1) || major > 5 {
		kTLSSupportAESGCM256 = true
		kTLSSupportTLS13 = true
	}

	if (major == 5 && minor >= 11) || major > 5 {
		kTLSSupportCHACHA20POLY1305 = true
	}
}

func (c *Conn) ReadFrom(r io.Reader) (n int64, err error) {
	if err := c.Handshake(); err != nil {
		return 0, err
	}
	return io.Copy(c.conn, r)
}

const maxBufferSize int64 = 4 * 1024 * 1024

func (c *Conn) writeToFile(f *os.File, remain int64) (written int64, err error, handled bool) {
	if remain <= 0 {
		return 0, nil, false
	}
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, nil, false
	}
	fi, err := f.Stat()
	if err != nil {
		return 0, nil, false
	}
	if offset+remain > fi.Size() {
		err = f.Truncate(offset + remain)
		if err != nil {
			Debugf("file truncate error: %s", err)
			return 0, nil, false
		}
	}

	// mmap must align on a page boundary
	// mmap from 0, use data from offset
	bytes, err := syscall.Mmap(int(f.Fd()), 0, int(offset+remain),
		syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return 0, nil, false
	}
	defer syscall.Munmap(bytes)

	bytes = bytes[offset : offset+remain]
	var (
		start = int64(0)
		end   = maxBufferSize
	)

	for {
		if end > remain {
			end = remain
		}
		//now := time.Now()
		n, err := c.Read(bytes[start:end])
		if err != nil {
			return start + int64(n), err, true
		}
		//log.Printf("read %d bytes, cost %dus", n, time.Since(now).Microseconds())
		start += int64(n)
		if start >= remain {
			break
		}

		end += int64(n)
	}
	return remain, nil, true
}

var maxSpliceSize int64 = 4 << 20

func (c *Conn) spliceToFile(f *os.File, remain int64) (written int64, err error, handled bool) {
	tcpConn, ok := c.conn.(*net.TCPConn)
	if !ok {
		return 0, nil, false
	}
	sc, err := tcpConn.SyscallConn()
	if err != nil {
		return 0, nil, false
	}
	fsc, err := f.SyscallConn()
	if err != nil {
		return 0, nil, false
	}

	var pipes [2]int
	if err := unix.Pipe(pipes[:]); err != nil {
		return 0, nil, false
	}

	prfd, pwfd := pipes[0], pipes[1]
	defer destroyTempPipe(prfd, pwfd)

	var (
		n = maxSpliceSize
		m int64
	)

	rerr := sc.Read(func(rfd uintptr) (done bool) {
		for {
			n = maxSpliceSize
			if n > remain {
				n = remain
			}
			// move tcp data to pipe
			// FIXME should not use unix.SPLICE_F_NONBLOCK, when use this flag, ktls will not advance socket buffer
			// refer: https://github.com/torvalds/linux/blob/v5.12/net/tls/tls_sw.c#L2021
			n, err = unix.Splice(int(rfd), nil, pwfd, nil, int(n), unix.SPLICE_F_MORE)
			remain -= n
			written += n
			if err == syscall.EAGAIN {
				// return false to wait data from connection
				err = nil
				return false
			}

			if err != nil {
				break
			}

			// move pipe data to file
			werr := fsc.Write(func(wfd uintptr) (done bool) {
			bump:
				m, err = unix.Splice(prfd, nil, int(wfd), nil, int(n),
					unix.SPLICE_F_MOVE|unix.SPLICE_F_MORE|unix.SPLICE_F_NONBLOCK)
				if err != nil {
					return true
				}
				if m < n {
					n -= m
					goto bump
				}
				return true
			})
			if err == nil {
				err = werr
			}
			if err != nil || remain <= 0 {
				break
			}
		}
		return true
	})
	if err == nil {
		err = rerr
	}
	return written, err, true
}

// destroyTempPipe destroys a temporary pipe.
func destroyTempPipe(prfd, pwfd int) error {
	err := syscall.Close(prfd)
	err1 := syscall.Close(pwfd)
	if err == nil {
		return err1
	}
	return err
}

func (c *Conn) WriteTo(w io.Writer) (n int64, err error) {
	if err := c.Handshake(); err != nil {
		return 0, err
	}

	if lw, ok := w.(*LimitedWriter); ok {
		if f, ok := lw.W.(*os.File); ok {
			n, err, handled := c.spliceToFile(f, lw.N)
			if handled {
				return n, err
			}
		}
	}

	// FIXME read at least one record for io.EOF and so on ?
	//if conn, ok := w.(*net.TCPConn); ok {
	//	buf := make([]byte, 16*1024)
	//	n, err := ktlsReadRecord(conn, buf)
	//	if err != nil {
	//		wn, _ := w.Write(buf[:n])
	//		return int64(wn), err
	//	}
	//	wn, err := w.Write(buf[:n])
	//	if err != nil {
	//		return int64(wn), err
	//	}
	//}
	return io.Copy(w, c.conn)
}

func (c *Conn) IsKTLSTXEnabled() bool {
	_, ok := c.out.cipher.(kTLSCipher)
	return ok
}

func (c *Conn) IsKTLSRXEnabled() bool {
	_, ok := c.in.cipher.(kTLSCipher)
	return ok
}

func (c *Conn) enableKernelTLS(cipherSuiteID uint16, inKey, outKey, inIV, outIV []byte) error {
	if !kTLSSupport {
		return nil
	}
	switch cipherSuiteID {
	// Kernel TLS 1.2
	case TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, TLS_RSA_WITH_AES_128_GCM_SHA256:
		if !kTLSSupportAESGCM128 {
			return nil
		}
		Debugln("try to enable kernel tls AES_128_GCM")
		return ktlsEnableAES(c, VersionTLS12, ktlsEnableAES128GCM, 16, inKey, outKey, inIV, outIV)
	case TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, TLS_RSA_WITH_AES_256_GCM_SHA384:
		if !kTLSSupportAESGCM256 {
			return nil
		}
		Debugln("try to enable kernel tls AES_256_GCM")
		return ktlsEnableAES(c, VersionTLS12, ktlsEnableAES256GCM, 32, inKey, outKey, inIV, outIV)
	case TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256, TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256:
		if !kTLSSupportCHACHA20POLY1305 {
			return nil
		}
		Debugln("try to enable kernel tls CHACHA20_POLY1305")
		return ktlsEnableCHACHA20(c, VersionTLS12, inKey, outKey, inIV, outIV)

	// Kernel TLS 1.3
	case TLS_AES_128_GCM_SHA256:
		if !kTLSSupportAESGCM128 {
			return nil
		}
		Debugln("try to enable kernel tls AES_128_GCM for tls 1.3")
		return ktlsEnableAES(c, VersionTLS13, ktlsEnableAES128GCM, 16, inKey, outKey, inIV, outIV)
	case TLS_AES_256_GCM_SHA384:
		if !kTLSSupportAESGCM256 {
			return nil
		}
		Debugln("try to enable kernel tls AES_256_GCM tls 1.3")
		return ktlsEnableAES(c, VersionTLS13, ktlsEnableAES256GCM, 32, inKey, outKey, inIV, outIV)
	case TLS_CHACHA20_POLY1305_SHA256:
		if !kTLSSupportCHACHA20POLY1305 {
			return nil
		}
		Debugln("try to enable kernel tls CHACHA20_POLY1305 for tls 1.3")
		return ktlsEnableCHACHA20(c, VersionTLS13, inKey, outKey, inIV, outIV)
	}
	return nil
}

func ktlsReadRecord(fd int, b []byte) (recordType, int, error) {
	// cmsg for record type
	buffer := make([]byte, syscall.CmsgSpace(1))
	cmsg := (*syscall.Cmsghdr)(unsafe.Pointer(&buffer[0]))
	cmsg.SetLen(syscall.CmsgLen(1))

	var iov syscall.Iovec
	iov.Base = &b[0]
	iov.SetLen(len(b))

	var msg syscall.Msghdr
	msg.Control = &buffer[0]
	msg.Controllen = cmsg.Len
	msg.Iov = &iov
	msg.Iovlen = 1

	var n int
	flags := 0
	n, err := recvmsg(uintptr(fd), &msg, flags)
	if err == syscall.EAGAIN {
		// data is not ready, goroutine will be parked
		return 0, n, err
	}
	// n should not be zero when err == nil
	if err == nil && n == 0 {
		err = io.EOF
	}

	if err != nil {
		Debugln("kTLS: recvmsg failed:", err)
		// fix bufio panic due to n == -1
		if n == -1 {
			n = 0
		}
		return 0, n, err
	}

	if n < 0 {
		return 0, 0, fmt.Errorf("unknown size received: %d", n)
	} else if n == 0 {
		return 0, 0, nil
	}

	if cmsg.Level != SOL_TLS {
		Debugf("kTLS: unsupported cmsg level: %d", cmsg.Level)
		return 0, 0, fmt.Errorf("unsupported cmsg level: %d", cmsg.Level)
	}
	if cmsg.Type != TLS_GET_RECORD_TYPE {
		Debugf("kTLS: unsupported cmsg type: %d", cmsg.Type)
		return 0, 0, fmt.Errorf("unsupported cmsg type: %d", cmsg.Type)
	}
	typ := recordType(buffer[syscall.SizeofCmsghdr])
	Debugf("kTLS: recvmsg, type: %d, payload len: %d", typ, n)
	return typ, n, nil
}

func ktlsReadDataFromRecord(fd int, b []byte) (int, error) {
	typ, n, err := ktlsReadRecord(fd, b)
	if err != nil {
		return n, err
	}
	switch typ {
	case recordTypeAlert:
		if n < 2 {
			return 0, fmt.Errorf("ktls alert payload too short")
		}
		if alert(b[1]) == alertCloseNotify {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("unsupported ktls alert type: %d", b[0])
	case recordTypeApplicationData:
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported ktls record type: %d", typ)
	}
}

func recvmsg(fd uintptr, msg *syscall.Msghdr, flags int) (n int, err error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_RECVMSG, fd, uintptr(unsafe.Pointer(msg)), uintptr(flags))
	n = int(r0)
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}

func sendmsg(fd uintptr, msg *syscall.Msghdr, flags int) (n int, err error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_SENDMSG, fd, uintptr(unsafe.Pointer(msg)), uintptr(flags))
	n = int(r0)
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}

// Do the interface allocations only once for common
// Errno values.
var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}

func ktlsSendCtrlMessage(fd int, typ recordType, b []byte) (int, error) {
	// cmsg for record type
	buffer := make([]byte, syscall.CmsgSpace(1))
	cmsg := (*syscall.Cmsghdr)(unsafe.Pointer(&buffer[0]))
	cmsg.SetLen(syscall.CmsgLen(1))
	buffer[syscall.SizeofCmsghdr] = byte(typ)
	cmsg.Level = SOL_TLS
	cmsg.Type = TLS_SET_RECORD_TYPE

	var iov syscall.Iovec
	iov.Base = &b[0]
	iov.SetLen(len(b))

	var msg syscall.Msghdr
	msg.Control = &buffer[0]
	msg.Controllen = cmsg.Len
	msg.Iov = &iov
	msg.Iovlen = 1

	var n int
	flags := 0
	n, err := sendmsg(uintptr(fd), &msg, flags)
	if err == syscall.EAGAIN {
		// data is not ready, goroutine will be parked
		return n, err
	}
	if err != nil {
		Debugln("kTLS: sendmsg failed:", err)
	}

	Debugf("kTLS: sendmsg, type: %d, payload len: %d", typ, len(b))
	return n, err
}
