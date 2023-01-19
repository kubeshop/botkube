// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.4
// source: executor.proto

package executor

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Config struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// RawYAML contains the Executor configuration in YAML definitions.
	RawYAML []byte `protobuf:"bytes,1,opt,name=rawYAML,proto3" json:"rawYAML,omitempty"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_executor_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_executor_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config.ProtoReflect.Descriptor instead.
func (*Config) Descriptor() ([]byte, []int) {
	return file_executor_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetRawYAML() []byte {
	if x != nil {
		return x.RawYAML
	}
	return nil
}

type ExecuteRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Commands represents the exact command that was specified by the user.
	Command string `protobuf:"bytes,1,opt,name=command,proto3" json:"command,omitempty"`
	// Configs is a list of Executor configurations specified by users.
	Configs []*Config `protobuf:"bytes,2,rep,name=configs,proto3" json:"configs,omitempty"`
}

func (x *ExecuteRequest) Reset() {
	*x = ExecuteRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_executor_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecuteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecuteRequest) ProtoMessage() {}

func (x *ExecuteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_executor_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecuteRequest.ProtoReflect.Descriptor instead.
func (*ExecuteRequest) Descriptor() ([]byte, []int) {
	return file_executor_proto_rawDescGZIP(), []int{1}
}

func (x *ExecuteRequest) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

func (x *ExecuteRequest) GetConfigs() []*Config {
	if x != nil {
		return x.Configs
	}
	return nil
}

type ExecuteResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data string `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ExecuteResponse) Reset() {
	*x = ExecuteResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_executor_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecuteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecuteResponse) ProtoMessage() {}

func (x *ExecuteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_executor_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecuteResponse.ProtoReflect.Descriptor instead.
func (*ExecuteResponse) Descriptor() ([]byte, []int) {
	return file_executor_proto_rawDescGZIP(), []int{2}
}

func (x *ExecuteResponse) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

type MetadataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// version is a version of a given plugin. It should follow the SemVer syntax.
	Version string `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	// description is a description of a given plugin.
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	// json_schema is a JSON schema of a given plugin.
	JsonSchema *JSONSchema `protobuf:"bytes,3,opt,name=json_schema,json=jsonSchema,proto3" json:"json_schema,omitempty"`
	// dependencies is a list of dependencies of a given plugin.
	Dependencies map[string]*Dependency `protobuf:"bytes,4,rep,name=dependencies,proto3" json:"dependencies,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *MetadataResponse) Reset() {
	*x = MetadataResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_executor_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetadataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetadataResponse) ProtoMessage() {}

func (x *MetadataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_executor_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetadataResponse.ProtoReflect.Descriptor instead.
func (*MetadataResponse) Descriptor() ([]byte, []int) {
	return file_executor_proto_rawDescGZIP(), []int{3}
}

func (x *MetadataResponse) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *MetadataResponse) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *MetadataResponse) GetJsonSchema() *JSONSchema {
	if x != nil {
		return x.JsonSchema
	}
	return nil
}

func (x *MetadataResponse) GetDependencies() map[string]*Dependency {
	if x != nil {
		return x.Dependencies
	}
	return nil
}

type JSONSchema struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// value is the string value of the JSON schema.
	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	// ref_url is the remote reference of the JSON schema.
	RefUrl string `protobuf:"bytes,2,opt,name=ref_url,json=refUrl,proto3" json:"ref_url,omitempty"`
}

func (x *JSONSchema) Reset() {
	*x = JSONSchema{}
	if protoimpl.UnsafeEnabled {
		mi := &file_executor_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JSONSchema) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JSONSchema) ProtoMessage() {}

func (x *JSONSchema) ProtoReflect() protoreflect.Message {
	mi := &file_executor_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JSONSchema.ProtoReflect.Descriptor instead.
func (*JSONSchema) Descriptor() ([]byte, []int) {
	return file_executor_proto_rawDescGZIP(), []int{4}
}

func (x *JSONSchema) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *JSONSchema) GetRefUrl() string {
	if x != nil {
		return x.RefUrl
	}
	return ""
}

type Dependency struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// urls is the map of URL of the dependency. The key is in format of "os/arch", such as "linux/amd64".
	Urls map[string]string `protobuf:"bytes,1,rep,name=urls,proto3" json:"urls,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Dependency) Reset() {
	*x = Dependency{}
	if protoimpl.UnsafeEnabled {
		mi := &file_executor_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Dependency) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Dependency) ProtoMessage() {}

func (x *Dependency) ProtoReflect() protoreflect.Message {
	mi := &file_executor_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Dependency.ProtoReflect.Descriptor instead.
func (*Dependency) Descriptor() ([]byte, []int) {
	return file_executor_proto_rawDescGZIP(), []int{5}
}

func (x *Dependency) GetUrls() map[string]string {
	if x != nil {
		return x.Urls
	}
	return nil
}

type HelpResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Help string `protobuf:"bytes,1,opt,name=help,proto3" json:"help,omitempty"`
}

func (x *HelpResponse) Reset() {
	*x = HelpResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_executor_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HelpResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HelpResponse) ProtoMessage() {}

func (x *HelpResponse) ProtoReflect() protoreflect.Message {
	mi := &file_executor_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HelpResponse.ProtoReflect.Descriptor instead.
func (*HelpResponse) Descriptor() ([]byte, []int) {
	return file_executor_proto_rawDescGZIP(), []int{6}
}

func (x *HelpResponse) GetHelp() string {
	if x != nil {
		return x.Help
	}
	return ""
}

var File_executor_proto protoreflect.FileDescriptor

var file_executor_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x08, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x6f, 0x72, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x22, 0x0a, 0x06, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x18, 0x0a, 0x07, 0x72, 0x61, 0x77, 0x59, 0x41, 0x4d, 0x4c, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x07, 0x72, 0x61, 0x77, 0x59, 0x41, 0x4d, 0x4c, 0x22, 0x56, 0x0a, 0x0e, 0x45,
	0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a,
	0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x2a, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x65, 0x78, 0x65, 0x63, 0x75,
	0x74, 0x6f, 0x72, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x73, 0x22, 0x25, 0x0a, 0x0f, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0xae, 0x02, 0x0a, 0x10, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73,
	0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b,
	0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x35, 0x0a, 0x0b, 0x6a,
	0x73, 0x6f, 0x6e, 0x5f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x14, 0x2e, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x6f, 0x72, 0x2e, 0x4a, 0x53, 0x4f, 0x4e,
	0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x52, 0x0a, 0x6a, 0x73, 0x6f, 0x6e, 0x53, 0x63, 0x68, 0x65,
	0x6d, 0x61, 0x12, 0x50, 0x0a, 0x0c, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x69,
	0x65, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x65, 0x78, 0x65, 0x63, 0x75,
	0x74, 0x6f, 0x72, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x69, 0x65,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0c, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e,
	0x63, 0x69, 0x65, 0x73, 0x1a, 0x55, 0x0a, 0x11, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e,
	0x63, 0x69, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x2a, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x65, 0x78, 0x65,
	0x63, 0x75, 0x74, 0x6f, 0x72, 0x2e, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x3b, 0x0a, 0x0a, 0x4a,
	0x53, 0x4f, 0x4e, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12,
	0x17, 0x0a, 0x07, 0x72, 0x65, 0x66, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x72, 0x65, 0x66, 0x55, 0x72, 0x6c, 0x22, 0x79, 0x0a, 0x0a, 0x44, 0x65, 0x70, 0x65,
	0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x12, 0x32, 0x0a, 0x04, 0x75, 0x72, 0x6c, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x6f, 0x72, 0x2e,
	0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x2e, 0x55, 0x72, 0x6c, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x75, 0x72, 0x6c, 0x73, 0x1a, 0x37, 0x0a, 0x09, 0x55, 0x72,
	0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x22, 0x22, 0x0a, 0x0c, 0x48, 0x65, 0x6c, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x65, 0x6c, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x68, 0x65, 0x6c, 0x70, 0x32, 0xc8, 0x01, 0x0a, 0x08, 0x45, 0x78, 0x65, 0x63,
	0x75, 0x74, 0x6f, 0x72, 0x12, 0x40, 0x0a, 0x07, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x12,
	0x18, 0x2e, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x6f, 0x72, 0x2e, 0x45, 0x78, 0x65, 0x63, 0x75,
	0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x65, 0x78, 0x65, 0x63,
	0x75, 0x74, 0x6f, 0x72, 0x2e, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x40, 0x0a, 0x08, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x1a, 0x2e, 0x65, 0x78, 0x65,
	0x63, 0x75, 0x74, 0x6f, 0x72, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x38, 0x0a, 0x04, 0x48, 0x65, 0x6c, 0x70,
	0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x16, 0x2e, 0x65, 0x78, 0x65, 0x63, 0x75,
	0x74, 0x6f, 0x72, 0x2e, 0x48, 0x65, 0x6c, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x00, 0x42, 0x12, 0x5a, 0x10, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x65, 0x78,
	0x65, 0x63, 0x75, 0x74, 0x6f, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_executor_proto_rawDescOnce sync.Once
	file_executor_proto_rawDescData = file_executor_proto_rawDesc
)

func file_executor_proto_rawDescGZIP() []byte {
	file_executor_proto_rawDescOnce.Do(func() {
		file_executor_proto_rawDescData = protoimpl.X.CompressGZIP(file_executor_proto_rawDescData)
	})
	return file_executor_proto_rawDescData
}

var file_executor_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_executor_proto_goTypes = []interface{}{
	(*Config)(nil),           // 0: executor.Config
	(*ExecuteRequest)(nil),   // 1: executor.ExecuteRequest
	(*ExecuteResponse)(nil),  // 2: executor.ExecuteResponse
	(*MetadataResponse)(nil), // 3: executor.MetadataResponse
	(*JSONSchema)(nil),       // 4: executor.JSONSchema
	(*Dependency)(nil),       // 5: executor.Dependency
	(*HelpResponse)(nil),     // 6: executor.HelpResponse
	nil,                      // 7: executor.MetadataResponse.DependenciesEntry
	nil,                      // 8: executor.Dependency.UrlsEntry
	(*emptypb.Empty)(nil),    // 9: google.protobuf.Empty
}
var file_executor_proto_depIdxs = []int32{
	0, // 0: executor.ExecuteRequest.configs:type_name -> executor.Config
	4, // 1: executor.MetadataResponse.json_schema:type_name -> executor.JSONSchema
	7, // 2: executor.MetadataResponse.dependencies:type_name -> executor.MetadataResponse.DependenciesEntry
	8, // 3: executor.Dependency.urls:type_name -> executor.Dependency.UrlsEntry
	5, // 4: executor.MetadataResponse.DependenciesEntry.value:type_name -> executor.Dependency
	1, // 5: executor.Executor.Execute:input_type -> executor.ExecuteRequest
	9, // 6: executor.Executor.Metadata:input_type -> google.protobuf.Empty
	9, // 7: executor.Executor.Help:input_type -> google.protobuf.Empty
	2, // 8: executor.Executor.Execute:output_type -> executor.ExecuteResponse
	3, // 9: executor.Executor.Metadata:output_type -> executor.MetadataResponse
	6, // 10: executor.Executor.Help:output_type -> executor.HelpResponse
	8, // [8:11] is the sub-list for method output_type
	5, // [5:8] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_executor_proto_init() }
func file_executor_proto_init() {
	if File_executor_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_executor_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Config); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_executor_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecuteRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_executor_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecuteResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_executor_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetadataResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_executor_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JSONSchema); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_executor_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Dependency); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_executor_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HelpResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_executor_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_executor_proto_goTypes,
		DependencyIndexes: file_executor_proto_depIdxs,
		MessageInfos:      file_executor_proto_msgTypes,
	}.Build()
	File_executor_proto = out.File
	file_executor_proto_rawDesc = nil
	file_executor_proto_goTypes = nil
	file_executor_proto_depIdxs = nil
}
