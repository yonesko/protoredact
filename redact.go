package protoredact

import (
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/descriptorpb"
)

var StdRedactor = Redactor{
	RedactingHandler: func(parent protoreflect.Value, fd protoreflect.FieldDescriptor) error {
		parent.Message().Clear(fd)
		return nil
	},
}

type Redactor struct {
	RedactingHandler func(parent protoreflect.Value, field protoreflect.FieldDescriptor) error
}

func (r Redactor) Redact(msg proto.Message, sensitiveFieldAnnotation *protoimpl.ExtensionInfo) error {
	return protorange.Range(msg.ProtoReflect(), func(p protopath.Values) error {
		fd := p.Path.Index(-1).FieldDescriptor()
		if isFieldSensetive(fd, p.Index(-1).Value, sensitiveFieldAnnotation) {
			parent := p.Index(-2)
			if parent.Value.IsValid() {
				err := r.RedactingHandler(parent.Value, fd)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func Redact(msg proto.Message, sensitiveFieldAnnotation *protoimpl.ExtensionInfo) error {
	return StdRedactor.Redact(msg, sensitiveFieldAnnotation)
}

func isFieldSensetive(fieldDescriptor protoreflect.FieldDescriptor, value protoreflect.Value, sensitiveFieldAnnotation *protoimpl.ExtensionInfo) bool {
	if fieldDescriptor == nil {
		return false
	}
	opts, ok := fieldDescriptor.Options().(*descriptorpb.FieldOptions)
	if !ok {
		return false
	}
	if !proto.HasExtension(opts, sensitiveFieldAnnotation) {
		return false
	}
	if fieldDescriptor.IsMap() {
		return handleMapType(fieldDescriptor, value, opts, sensitiveFieldAnnotation)
	}
	return true
}

/*
if GetMapKeysToRedact is empty, hide the whole field, otherwise hide only specified keys
*/
func handleMapType(fd protoreflect.FieldDescriptor, value protoreflect.Value, opts *descriptorpb.FieldOptions, sensitiveFieldAnnotation *protoimpl.ExtensionInfo) bool {
	if !fd.IsMap() || !value.Map().IsValid() {
		return false
	}
	ext, ok := proto.GetExtension(opts, sensitiveFieldAnnotation).(interface {
		GetMapKeysToRedact() []string
	})
	if !ok {
		return false
	}
	keysToHide := lo.SliceToMap(ext.GetMapKeysToRedact(), func(item string) (string, bool) {
		return item, true
	})
	if len(keysToHide) == 0 {
		return true
	}

	valueMap := value.Map()
	valueMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
		if keysToHide[key.String()] {
			valueMap.Set(key, valueMap.NewValue())
		}
		return true
	})
	return false
}
