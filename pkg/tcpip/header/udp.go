// Copyright 2018 The gVisor Authors.
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

package header

import (
	"encoding/binary"
	"math"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
)

const (
	udpSrcPort  = 0
	udpDstPort  = 2
	udpLength   = 4
	udpChecksum = 6
)

const (
	// UDPMaximumPacketSize is the largest possible UDP packet.
	UDPMaximumPacketSize = 0xffff
)

// UDPFields contains the fields of a UDP packet. It is used to describe the
// fields of a packet that needs to be encoded.
type UDPFields struct {
	// SrcPort is the "source port" field of a UDP packet.
	SrcPort uint16

	// DstPort is the "destination port" field of a UDP packet.
	DstPort uint16

	// Length is the "length" field of a UDP packet.
	Length uint16

	// Checksum is the "checksum" field of a UDP packet.
	Checksum uint16
}

// UDP represents a UDP header stored in a byte array.
type UDP []byte

const (
	// UDPMinimumSize is the minimum size of a valid UDP packet.
	UDPMinimumSize = 8

	// UDPMaximumSize is the maximum size of a valid UDP packet. The length field
	// in the UDP header is 16 bits as per RFC 768.
	UDPMaximumSize = math.MaxUint16

	// UDPProtocolNumber is UDP's transport protocol number.
	UDPProtocolNumber tcpip.TransportProtocolNumber = 17
)

// SourcePort returns the "source port" field of the udp header.
func (b UDP) SourcePort() uint16 {
	return binary.BigEndian.Uint16(b[udpSrcPort:])
}

// DestinationPort returns the "destination port" field of the udp header.
func (b UDP) DestinationPort() uint16 {
	return binary.BigEndian.Uint16(b[udpDstPort:])
}

// Length returns the "length" field of the udp header.
func (b UDP) Length() uint16 {
	return binary.BigEndian.Uint16(b[udpLength:])
}

// Payload returns the data contained in the UDP datagram.
func (b UDP) Payload() []byte {
	return b[UDPMinimumSize:]
}

// Checksum returns the "checksum" field of the udp header.
func (b UDP) Checksum() uint16 {
	return binary.BigEndian.Uint16(b[udpChecksum:])
}

// SetSourcePort sets the "source port" field of the udp header.
func (b UDP) SetSourcePort(port uint16) {
	binary.BigEndian.PutUint16(b[udpSrcPort:], port)
}

// SetDestinationPort sets the "destination port" field of the udp header.
func (b UDP) SetDestinationPort(port uint16) {
	binary.BigEndian.PutUint16(b[udpDstPort:], port)
}

// SetChecksum sets the "checksum" field of the udp header.
func (b UDP) SetChecksum(checksum uint16) {
	binary.BigEndian.PutUint16(b[udpChecksum:], checksum)
}

// SetLength sets the "length" field of the udp header.
func (b UDP) SetLength(length uint16) {
	binary.BigEndian.PutUint16(b[udpLength:], length)
}

// CalculateChecksum calculates the checksum of the udp packet, given the
// checksum of the network-layer pseudo-header and the checksum of the payload.
func (b UDP) CalculateChecksum(partialChecksum uint16) uint16 {
	// Calculate the rest of the checksum.
	return Checksum(b[:UDPMinimumSize], partialChecksum)
}

// IsChecksumValid performs checksum validation.
func (b UDP) IsChecksumValid(src, dst tcpip.Address, netProto tcpip.NetworkProtocolNumber, data buffer.VectorisedView) bool {
	// On IPv4, UDP checksum is optional, and a zero value means the transmitter
	// omitted the checksum generation, as per RFC 768:
	//
	//   An all zero transmitted checksum value means that the transmitter
	//   generated  no checksum  (for debugging or for higher level protocols that
	//   don't care).
	//
	// On IPv6, UDP checksum is not optional, as per RFC 2460 Section 8.1:
	//
	//   Unlike IPv4, when UDP packets are originated by an IPv6 node, the UDP
	//   checksum is not optional.
	if b.Checksum() == 0 && netProto == IPv4ProtocolNumber {
		return true
	}
	xsum := PseudoHeaderChecksum(UDPProtocolNumber, dst, src, b.Length())
	for _, v := range data.Views() {
		xsum = Checksum(v, xsum)
	}
	return b.CalculateChecksum(xsum) == 0xffff
}

// Encode encodes all the fields of the udp header.
func (b UDP) Encode(u *UDPFields) {
	binary.BigEndian.PutUint16(b[udpSrcPort:], u.SrcPort)
	binary.BigEndian.PutUint16(b[udpDstPort:], u.DstPort)
	binary.BigEndian.PutUint16(b[udpLength:], u.Length)
	binary.BigEndian.PutUint16(b[udpChecksum:], u.Checksum)
}
