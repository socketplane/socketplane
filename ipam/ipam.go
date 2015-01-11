package ipam

import (
	"math"
	"net"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

// Simple IPv4 IPAM solution using Consul Distributed KV store
// Key = subnet, Value = Bit Array of available ip-addresses in a given subnet

const dataStore = "ipam"

func Request(subnet net.IPNet) net.IP {
	bits := bitCount(subnet)
	bc := int(bits / 8)
	partial := int(math.Mod(bits, float64(8)))

	if partial != 0 {
		bc += 1
	}
	addrArray, _, ok := ecc.Get(dataStore, subnet.String())
	currVal := make([]byte, len(addrArray))
	copy(currVal, addrArray)
	if !ok {
		addrArray = make([]byte, bc)
	}
	pos := testAndSetBit(addrArray)
	eccerr := ecc.Put(dataStore, subnet.String(), addrArray, currVal)
	if eccerr == ecc.OUTDATED {
		return Request(subnet)
	}
	return getIP(subnet, pos)
}

func Release(address net.IP, subnet net.IPNet) bool {
	addrArray, _, ok := ecc.Get(dataStore, subnet.String())
	currVal := make([]byte, len(addrArray))
	copy(currVal, addrArray)
	if !ok {
		return false
	}
	pos := getBitPosition(address, subnet)
	clearBit(addrArray, pos-1)
	eccerr := ecc.Put(dataStore, subnet.String(), addrArray, currVal)
	if eccerr == ecc.OUTDATED {
		return Release(address, subnet)
	}
	return true
}

func getBitPosition(address net.IP, subnet net.IPNet) uint {
	mask, size := subnet.Mask.Size()
	if address.To4() != nil {
		address = address.To4()
	}
	tb := size / 8
	byteCount := (size - mask) / 8
	bitCount := (size - mask) % 8
	pos := uint(0)

	for i := 0; i <= byteCount; i++ {
		maskLen := 0xFF
		if i == byteCount {
			if bitCount != 0 {
				maskLen = int(math.Pow(2, float64(bitCount))) - 1
			} else {
				maskLen = 0
			}
		}
		pos += (uint(address[tb-i-1]) & uint(0xFF&maskLen)) << uint(8*i)
	}
	return pos
}

// Given Subnet of interest and free bit position, this method returns the corresponding ip address
// This method is functional and tested. Refer to ipam_test.go But can be improved

func getIP(subnet net.IPNet, pos uint) net.IP {
	retAddr := make([]byte, len(subnet.IP))
	copy(retAddr, subnet.IP)

	mask, _ := subnet.Mask.Size()
	var tb, byteCount, bitCount int
	if subnet.IP.To4() != nil {
		tb = 4
		byteCount = (32 - mask) / 8
		bitCount = (32 - mask) % 8
	} else {
		tb = 16
		byteCount = (128 - mask) / 8
		bitCount = (128 - mask) % 8
	}

	for i := 0; i <= byteCount; i++ {
		maskLen := 0xFF
		if i == byteCount {
			if bitCount != 0 {
				maskLen = int(math.Pow(2, float64(bitCount))) - 1
			} else {
				maskLen = 0
			}
		}
		masked := pos & uint((0xFF&maskLen)<<uint(8*i))
		retAddr[tb-i-1] |= byte(masked >> uint(8*i))
	}
	return net.IP(retAddr)
}

func bitCount(addr net.IPNet) float64 {
	mask, _ := addr.Mask.Size()
	if addr.IP.To4() != nil {
		return math.Pow(2, float64(32-mask))
	} else {
		return math.Pow(2, float64(128-mask))
	}
}

func setBit(a []byte, k uint) {
	a[k/8] |= 1 << (k % 8)
}

func clearBit(a []byte, k uint) {
	a[k/8] &= ^(1 << (k % 8))
}

func testBit(a []byte, k uint) bool {
	return ((a[k/8] & (1 << (k % 8))) != 0)
}

func testAndSetBit(a []byte) uint {
	var i uint
	for i = uint(0); i < uint(len(a)*8); i++ {
		if !testBit(a, i) {
			setBit(a, i)
			return i + 1
		}
	}
	return i
}
