// +build darwin
package gofi

import (
	"errors"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ioctlBIOCSETIF is an ioctl command used for setting the network interface
// on a BPF device.
const ioctlBIOCSETIF = 0x8020426c

// ioctlBIOCSDLT is an ioctl command used for setting the data-link type
// on a BPF device.
const ioctlBIOCSDLT = 0x80044278

// dltIEEE802_11_RADIO is a data-link type for ioctlBIOCSDLT.
// Read more here:
// http://www.opensource.apple.com/source/tcpdump/tcpdump-16/tcpdump/ieee802_11_radio.h
const dltIEEE802_11_RADIO = 127

// dltIEEE802_11 is a data-link type for ioctlBIOCSDLT.
// This data-link type captures 802.11 headers with no extra info.
const dltIEEE802_11 = 105

type bpfHandle struct {
	fd           int
	dataLinkType int
}

func newBpfHandle() (*bpfHandle, error) {
	res, err := unix.Open("/dev/bpf", unix.O_RDWR, 0)
	if err == nil {
		return &bpfHandle{fd: res}, nil
	} else if err == unix.EACCES {
		return nil, errors.New("permissions denied for: /dev/bpf")
	}
	i := 0
	for {
		devName := "/dev/bpf" + strconv.Itoa(i)
		res, err := unix.Open(devName, unix.O_RDWR, 0)
		if err == nil {
			return &bpfHandle{fd: res}, nil
		} else if err == unix.EACCES {
			return nil, errors.New("permissions denied for: " + devName)
		} else if err != unix.EBUSY {
			return nil, err
		}
		i++
	}
}

// SetInterface assigns an interface name to the BPF handle.
func (b *bpfHandle) SetInterface(name string) error {
	data := make([]byte, 100)
	copy(data, []byte(name))
	if ok, err := b.ioctlWithData(ioctlBIOCSETIF, data); ok {
		return nil
	} else if err == unix.ENXIO {
		return errors.New("no such device: " + name)
	} else if err == unix.ENETDOWN {
		return errors.New("interface is down: " + name)
	} else {
		return err
	}
}

// SetupDataLink switches to a data-link type that provides raw 802.11 headers.
// If no 802.11 DLT is supported on the interface, this returns an error.
func (b *bpfHandle) SetupDataLink() error {
	numBuf := make([]byte, 16)
	numBuf[0] = dltIEEE802_11_RADIO
	if ok, _ := b.ioctlWithData(ioctlBIOCSDLT, numBuf); ok {
		b.dataLinkType = dltIEEE802_11_RADIO
		return nil
	}
	numBuf[0] = dltIEEE802_11
	if ok, _ := b.ioctlWithData(ioctlBIOCSDLT, numBuf); ok {
		b.dataLinkType = dltIEEE802_11
		return nil
	}
	return errors.New("could not use an 802.11 data-link type")
}

func (b *bpfHandle) ioctlWithData(command int, data []byte) (ok bool, err syscall.Errno) {
	_, _, err = unix.Syscall(unix.SYS_IOCTL, uintptr(b.fd), uintptr(command),
		uintptr(unsafe.Pointer(&data[0])))
	if err != 0 {
		return
	} else {
		return true, 0
	}
}
