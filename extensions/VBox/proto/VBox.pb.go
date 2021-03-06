// Code generated by protoc-gen-go. DO NOT EDIT.
// source: VBox.proto

package proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type VBox_PXE int32

const (
	VBox_NONE VBox_PXE = 0
	VBox_WAIT VBox_PXE = 1
	VBox_INIT VBox_PXE = 2
	VBox_COMP VBox_PXE = 3
)

var VBox_PXE_name = map[int32]string{
	0: "NONE",
	1: "WAIT",
	2: "INIT",
	3: "COMP",
}
var VBox_PXE_value = map[string]int32{
	"NONE": 0,
	"WAIT": 1,
	"INIT": 2,
	"COMP": 3,
}

func (x VBox_PXE) String() string {
	return proto.EnumName(VBox_PXE_name, int32(x))
}
func (VBox_PXE) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_VBox_4b44f594c8967582, []int{0, 0}
}

type VBox struct {
	ApiServer            string   `protobuf:"bytes,1,opt,name=api_server,json=apiServer,proto3" json:"api_server,omitempty"`
	Name                 string   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Uuid                 string   `protobuf:"bytes,3,opt,name=uuid,proto3" json:"uuid,omitempty"`
	Pxe                  VBox_PXE `protobuf:"varint,4,opt,name=pxe,proto3,enum=proto.VBox_PXE" json:"pxe,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VBox) Reset()         { *m = VBox{} }
func (m *VBox) String() string { return proto.CompactTextString(m) }
func (*VBox) ProtoMessage()    {}
func (*VBox) Descriptor() ([]byte, []int) {
	return fileDescriptor_VBox_4b44f594c8967582, []int{0}
}
func (m *VBox) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VBox.Unmarshal(m, b)
}
func (m *VBox) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VBox.Marshal(b, m, deterministic)
}
func (dst *VBox) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VBox.Merge(dst, src)
}
func (m *VBox) XXX_Size() int {
	return xxx_messageInfo_VBox.Size(m)
}
func (m *VBox) XXX_DiscardUnknown() {
	xxx_messageInfo_VBox.DiscardUnknown(m)
}

var xxx_messageInfo_VBox proto.InternalMessageInfo

func (m *VBox) GetApiServer() string {
	if m != nil {
		return m.ApiServer
	}
	return ""
}

func (m *VBox) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *VBox) GetUuid() string {
	if m != nil {
		return m.Uuid
	}
	return ""
}

func (m *VBox) GetPxe() VBox_PXE {
	if m != nil {
		return m.Pxe
	}
	return VBox_NONE
}

func init() {
	proto.RegisterType((*VBox)(nil), "proto.VBox")
	proto.RegisterEnum("proto.VBox_PXE", VBox_PXE_name, VBox_PXE_value)
}

func init() { proto.RegisterFile("VBox.proto", fileDescriptor_VBox_4b44f594c8967582) }

var fileDescriptor_VBox_4b44f594c8967582 = []byte{
	// 171 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x0a, 0x73, 0xca, 0xaf,
	0xd0, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x53, 0x4a, 0xf3, 0x19, 0xb9, 0x58, 0x40,
	0xa2, 0x42, 0xb2, 0x5c, 0x5c, 0x89, 0x05, 0x99, 0xf1, 0xc5, 0xa9, 0x45, 0x65, 0xa9, 0x45, 0x12,
	0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x9c, 0x89, 0x05, 0x99, 0xc1, 0x60, 0x01, 0x21, 0x21, 0x2e,
	0x96, 0xbc, 0xc4, 0xdc, 0x54, 0x09, 0x26, 0xb0, 0x04, 0x98, 0x0d, 0x12, 0x2b, 0x2d, 0xcd, 0x4c,
	0x91, 0x60, 0x86, 0x88, 0x81, 0xd8, 0x42, 0x8a, 0x5c, 0xcc, 0x05, 0x15, 0xa9, 0x12, 0x2c, 0x0a,
	0x8c, 0x1a, 0x7c, 0x46, 0xfc, 0x10, 0xbb, 0xf4, 0xc0, 0xd6, 0x06, 0x44, 0xb8, 0x06, 0x81, 0xe4,
	0x94, 0x74, 0xb9, 0x98, 0x03, 0x22, 0x5c, 0x85, 0x38, 0xb8, 0x58, 0xfc, 0xfc, 0xfd, 0x5c, 0x05,
	0x18, 0x40, 0xac, 0x70, 0x47, 0xcf, 0x10, 0x01, 0x46, 0x10, 0xcb, 0xd3, 0xcf, 0x33, 0x44, 0x80,
	0x09, 0xc4, 0x72, 0xf6, 0xf7, 0x0d, 0x10, 0x60, 0x4e, 0x62, 0x03, 0x9b, 0x61, 0x0c, 0x08, 0x00,
	0x00, 0xff, 0xff, 0x8c, 0xfe, 0xa6, 0x54, 0xbd, 0x00, 0x00, 0x00,
}
