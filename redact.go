package protoredact

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/descriptorpb"
	"os"
	"sync"
)

var (
	clearFunc = func(parent protoreflect.Value, fd protoreflect.FieldDescriptor) error {
		parent.Message().Clear(fd)
		return nil
	}
	o sync.Once
)

type Redactor struct {
	SensitiveFieldAnnotation *protoimpl.ExtensionInfo
	RedactingHandler         func(parent protoreflect.Value, field protoreflect.FieldDescriptor) error
}

func (r Redactor) Redact(msg proto.Message) error {
	o.Do(func() {
		fmt.Println(os.Environ())
	})
	if r.SensitiveFieldAnnotation == nil || r.RedactingHandler == nil {
		return nil
	}
	return protorange.Range(msg.ProtoReflect(), func(p protopath.Values) error {
		fd := p.Path.Index(-1).FieldDescriptor()
		if isFieldSensetive(fd, p.Index(-1).Value, r.SensitiveFieldAnnotation) {
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
	return Redactor{RedactingHandler: clearFunc, SensitiveFieldAnnotation: sensitiveFieldAnnotation}.Redact(msg)
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
	keysToHide := associate(ext.GetMapKeysToRedact(), func(item string) (string, bool) {
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

func associate[T any, K comparable, V any](collection []T, transform func(item T) (K, V)) map[K]V {
	result := make(map[K]V, len(collection))

	for _, t := range collection {
		k, v := transform(t)
		result[k] = v
	}

	return result
}
