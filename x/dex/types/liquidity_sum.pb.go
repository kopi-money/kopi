// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: kopi/dex/liquidity_sum.proto

package types

import (
	cosmossdk_io_math "cosmossdk.io/math"
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type LiquiditySum struct {
	Denom  string                `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty"`
	Amount cosmossdk_io_math.Int `protobuf:"bytes,2,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
}

func (m *LiquiditySum) Reset()         { *m = LiquiditySum{} }
func (m *LiquiditySum) String() string { return proto.CompactTextString(m) }
func (*LiquiditySum) ProtoMessage()    {}
func (*LiquiditySum) Descriptor() ([]byte, []int) {
	return fileDescriptor_7d4c2aa311a7102a, []int{0}
}
func (m *LiquiditySum) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LiquiditySum) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LiquiditySum.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *LiquiditySum) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LiquiditySum.Merge(m, src)
}
func (m *LiquiditySum) XXX_Size() int {
	return m.Size()
}
func (m *LiquiditySum) XXX_DiscardUnknown() {
	xxx_messageInfo_LiquiditySum.DiscardUnknown(m)
}

var xxx_messageInfo_LiquiditySum proto.InternalMessageInfo

func (m *LiquiditySum) GetDenom() string {
	if m != nil {
		return m.Denom
	}
	return ""
}

func init() {
	proto.RegisterType((*LiquiditySum)(nil), "kopi.dex.LiquiditySum")
}

func init() { proto.RegisterFile("kopi/dex/liquidity_sum.proto", fileDescriptor_7d4c2aa311a7102a) }

var fileDescriptor_7d4c2aa311a7102a = []byte{
	// 218 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0xc9, 0xce, 0x2f, 0xc8,
	0xd4, 0x4f, 0x49, 0xad, 0xd0, 0xcf, 0xc9, 0x2c, 0x2c, 0xcd, 0x4c, 0xc9, 0x2c, 0xa9, 0x8c, 0x2f,
	0x2e, 0xcd, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x00, 0xc9, 0xea, 0xa5, 0xa4, 0x56,
	0x48, 0x89, 0xa4, 0xe7, 0xa7, 0xe7, 0x83, 0x05, 0xf5, 0x41, 0x2c, 0x88, 0xbc, 0x52, 0x34, 0x17,
	0x8f, 0x0f, 0x4c, 0x5b, 0x70, 0x69, 0xae, 0x90, 0x08, 0x17, 0x6b, 0x4a, 0x6a, 0x5e, 0x7e, 0xae,
	0x04, 0xa3, 0x02, 0xa3, 0x06, 0x67, 0x10, 0x84, 0x23, 0x64, 0xca, 0xc5, 0x96, 0x98, 0x9b, 0x5f,
	0x9a, 0x57, 0x22, 0xc1, 0xa4, 0xc0, 0xa8, 0xc1, 0xe3, 0x24, 0x7b, 0xe2, 0x9e, 0x3c, 0xc3, 0xad,
	0x7b, 0xf2, 0xa2, 0xc9, 0xf9, 0xc5, 0xb9, 0xf9, 0xc5, 0xc5, 0x29, 0xd9, 0x7a, 0x99, 0xf9, 0xfa,
	0xb9, 0x89, 0x25, 0x19, 0x7a, 0x9e, 0x79, 0x25, 0x41, 0x50, 0xc5, 0x4e, 0x0e, 0x27, 0x1e, 0xc9,
	0x31, 0x5e, 0x78, 0x24, 0xc7, 0xf8, 0xe0, 0x91, 0x1c, 0xe3, 0x84, 0xc7, 0x72, 0x0c, 0x17, 0x1e,
	0xcb, 0x31, 0xdc, 0x78, 0x2c, 0xc7, 0x10, 0xa5, 0x96, 0x9e, 0x59, 0x92, 0x51, 0x9a, 0xa4, 0x97,
	0x9c, 0x9f, 0xab, 0x0f, 0x72, 0xa1, 0x6e, 0x6e, 0x7e, 0x5e, 0x6a, 0x25, 0x98, 0xa9, 0x5f, 0x01,
	0xf6, 0x4c, 0x49, 0x65, 0x41, 0x6a, 0x71, 0x12, 0x1b, 0xd8, 0x95, 0xc6, 0x80, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xe8, 0x9f, 0x96, 0x0b, 0xe5, 0x00, 0x00, 0x00,
}

func (m *LiquiditySum) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LiquiditySum) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LiquiditySum) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Amount.Size()
		i -= size
		if _, err := m.Amount.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintLiquiditySum(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Denom) > 0 {
		i -= len(m.Denom)
		copy(dAtA[i:], m.Denom)
		i = encodeVarintLiquiditySum(dAtA, i, uint64(len(m.Denom)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintLiquiditySum(dAtA []byte, offset int, v uint64) int {
	offset -= sovLiquiditySum(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *LiquiditySum) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Denom)
	if l > 0 {
		n += 1 + l + sovLiquiditySum(uint64(l))
	}
	l = m.Amount.Size()
	n += 1 + l + sovLiquiditySum(uint64(l))
	return n
}

func sovLiquiditySum(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozLiquiditySum(x uint64) (n int) {
	return sovLiquiditySum(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *LiquiditySum) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLiquiditySum
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: LiquiditySum: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LiquiditySum: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquiditySum
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLiquiditySum
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLiquiditySum
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Denom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquiditySum
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthLiquiditySum
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthLiquiditySum
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Amount.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLiquiditySum(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthLiquiditySum
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipLiquiditySum(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowLiquiditySum
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowLiquiditySum
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowLiquiditySum
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthLiquiditySum
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupLiquiditySum
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthLiquiditySum
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthLiquiditySum        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowLiquiditySum          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupLiquiditySum = fmt.Errorf("proto: unexpected end of group")
)
