// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: kopi/tokenfactory/genesis.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
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

// GenesisState defines the tokenfactory module's genesis state.
type GenesisState struct {
	// params defines all the parameters of the module.
	Params        Params         `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
	FactoryDenoms []FactoryDenom `protobuf:"bytes,2,rep,name=factory_denoms,json=factoryDenoms,proto3" json:"factory_denoms" yaml:"factory_denoms"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_b8412730f80620c1, []int{0}
}
func (m *GenesisState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenesisState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenesisState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenesisState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenesisState.Merge(m, src)
}
func (m *GenesisState) XXX_Size() int {
	return m.Size()
}
func (m *GenesisState) XXX_DiscardUnknown() {
	xxx_messageInfo_GenesisState.DiscardUnknown(m)
}

var xxx_messageInfo_GenesisState proto.InternalMessageInfo

func (m *GenesisState) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

func (m *GenesisState) GetFactoryDenoms() []FactoryDenom {
	if m != nil {
		return m.FactoryDenoms
	}
	return nil
}

type FactoryDenom struct {
	Denom string `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty" yaml:"denom"`
	Admin string `protobuf:"bytes,2,opt,name=admin,proto3" json:"admin,omitempty" yaml:"admin"`
}

func (m *FactoryDenom) Reset()         { *m = FactoryDenom{} }
func (m *FactoryDenom) String() string { return proto.CompactTextString(m) }
func (*FactoryDenom) ProtoMessage()    {}
func (*FactoryDenom) Descriptor() ([]byte, []int) {
	return fileDescriptor_b8412730f80620c1, []int{1}
}
func (m *FactoryDenom) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *FactoryDenom) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_FactoryDenom.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *FactoryDenom) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FactoryDenom.Merge(m, src)
}
func (m *FactoryDenom) XXX_Size() int {
	return m.Size()
}
func (m *FactoryDenom) XXX_DiscardUnknown() {
	xxx_messageInfo_FactoryDenom.DiscardUnknown(m)
}

var xxx_messageInfo_FactoryDenom proto.InternalMessageInfo

func (m *FactoryDenom) GetDenom() string {
	if m != nil {
		return m.Denom
	}
	return ""
}

func (m *FactoryDenom) GetAdmin() string {
	if m != nil {
		return m.Admin
	}
	return ""
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "kopi.tokenfactory.GenesisState")
	proto.RegisterType((*FactoryDenom)(nil), "kopi.tokenfactory.FactoryDenom")
}

func init() { proto.RegisterFile("kopi/tokenfactory/genesis.proto", fileDescriptor_b8412730f80620c1) }

var fileDescriptor_b8412730f80620c1 = []byte{
	// 318 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0xcf, 0xce, 0x2f, 0xc8,
	0xd4, 0x2f, 0xc9, 0xcf, 0x4e, 0xcd, 0x4b, 0x4b, 0x4c, 0x2e, 0xc9, 0x2f, 0xaa, 0xd4, 0x4f, 0x4f,
	0xcd, 0x4b, 0x2d, 0xce, 0x2c, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x04, 0x29, 0xd0,
	0x43, 0x56, 0x20, 0x25, 0x98, 0x98, 0x9b, 0x99, 0x97, 0xaf, 0x0f, 0x26, 0x21, 0xaa, 0xa4, 0x44,
	0xd2, 0xf3, 0xd3, 0xf3, 0xc1, 0x4c, 0x7d, 0x10, 0x0b, 0x2a, 0x2a, 0x87, 0x69, 0x78, 0x41, 0x62,
	0x51, 0x62, 0x2e, 0xd4, 0x6c, 0xa5, 0xcd, 0x8c, 0x5c, 0x3c, 0xee, 0x10, 0xdb, 0x82, 0x4b, 0x12,
	0x4b, 0x52, 0x85, 0x6c, 0xb8, 0xd8, 0x20, 0x0a, 0x24, 0x18, 0x15, 0x18, 0x35, 0xb8, 0x8d, 0x24,
	0xf5, 0x30, 0x6c, 0xd7, 0x0b, 0x00, 0x2b, 0x70, 0xe2, 0x3c, 0x71, 0x4f, 0x9e, 0x61, 0xc5, 0xf3,
	0x0d, 0x5a, 0x8c, 0x41, 0x50, 0x3d, 0x42, 0xa9, 0x5c, 0x7c, 0x50, 0x45, 0xf1, 0x29, 0xa9, 0x79,
	0xf9, 0xb9, 0xc5, 0x12, 0x4c, 0x0a, 0xcc, 0x1a, 0xdc, 0x46, 0xf2, 0x58, 0x4c, 0x71, 0x83, 0xd0,
	0x2e, 0x20, 0x75, 0x4e, 0xb2, 0x20, 0xb3, 0x3e, 0xdd, 0x93, 0x17, 0xad, 0x4c, 0xcc, 0xcd, 0xb1,
	0x52, 0x42, 0x35, 0x44, 0x29, 0x88, 0x37, 0x0d, 0x49, 0x71, 0xb1, 0x52, 0x0a, 0x17, 0x0f, 0xb2,
	0x6e, 0x21, 0x35, 0x2e, 0x56, 0xb0, 0x4a, 0xb0, 0x9b, 0x39, 0x9d, 0x04, 0x3e, 0xdd, 0x93, 0xe7,
	0x81, 0x18, 0x04, 0x16, 0x56, 0x0a, 0x82, 0x48, 0x83, 0xd4, 0x25, 0xa6, 0xe4, 0x66, 0xe6, 0x49,
	0x30, 0xa1, 0xab, 0x03, 0x0b, 0x2b, 0x05, 0x41, 0xa4, 0xad, 0x58, 0x5e, 0x2c, 0x90, 0x67, 0x74,
	0xf2, 0x3c, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c,
	0x96, 0x63, 0xb8, 0xf0, 0x58, 0x8e, 0xe1, 0xc6, 0x63, 0x39, 0x86, 0x28, 0xfd, 0xf4, 0xcc, 0x92,
	0x8c, 0xd2, 0x24, 0xbd, 0xe4, 0xfc, 0x5c, 0x7d, 0x90, 0xc7, 0x74, 0x73, 0xf3, 0xf3, 0x52, 0x2b,
	0xc1, 0x4c, 0xfd, 0x0a, 0xd4, 0xd0, 0x2e, 0xa9, 0x2c, 0x48, 0x2d, 0x4e, 0x62, 0x03, 0x87, 0xb6,
	0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0xd6, 0xb8, 0x78, 0x1d, 0xec, 0x01, 0x00, 0x00,
}

func (this *FactoryDenom) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*FactoryDenom)
	if !ok {
		that2, ok := that.(FactoryDenom)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Denom != that1.Denom {
		return false
	}
	if this.Admin != that1.Admin {
		return false
	}
	return true
}
func (m *GenesisState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenesisState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenesisState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.FactoryDenoms) > 0 {
		for iNdEx := len(m.FactoryDenoms) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.FactoryDenoms[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenesis(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *FactoryDenom) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *FactoryDenom) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *FactoryDenom) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Admin) > 0 {
		i -= len(m.Admin)
		copy(dAtA[i:], m.Admin)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.Admin)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Denom) > 0 {
		i -= len(m.Denom)
		copy(dAtA[i:], m.Denom)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.Denom)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintGenesis(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenesis(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovGenesis(uint64(l))
	if len(m.FactoryDenoms) > 0 {
		for _, e := range m.FactoryDenoms {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	return n
}

func (m *FactoryDenom) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Denom)
	if l > 0 {
		n += 1 + l + sovGenesis(uint64(l))
	}
	l = len(m.Admin)
	if l > 0 {
		n += 1 + l + sovGenesis(uint64(l))
	}
	return n
}

func sovGenesis(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenesis(x uint64) (n int) {
	return sovGenesis(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FactoryDenoms", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FactoryDenoms = append(m.FactoryDenoms, FactoryDenom{})
			if err := m.FactoryDenoms[len(m.FactoryDenoms)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func (m *FactoryDenom) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: FactoryDenom: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: FactoryDenom: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Denom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Admin", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Admin = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func skipGenesis(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
				return 0, ErrInvalidLengthGenesis
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenesis
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenesis
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenesis        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenesis          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenesis = fmt.Errorf("proto: unexpected end of group")
)