package protoredact

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/yonesko/protoredact/testproto/go/testproto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"testing"
)

func TestRedactProto_Clear(t *testing.T) {
	t.Parallel()
	redactor := Redactor{
		RedactingHandler: func(parent protoreflect.Value, fd protoreflect.FieldDescriptor) error {
			parent.Message().Clear(fd)
			return nil
		},
		SensitiveFieldAnnotation: testproto.E_SensitiveData,
	}

	type args struct {
		message proto.Message
	}
	tests := []struct {
		name    string
		args    args
		want    proto.Message
		wantErr bool
	}{
		{
			name: "empty message",
			args: args{
				message: &testproto.WithAllFieldTypes{},
			},
			want:    &testproto.WithAllFieldTypes{},
			wantErr: false,
		},
		{
			name: "message with no sensetive",
			args: args{
				message: &testproto.WithAllFieldTypes{FieldInt64: 418, Enum1: testproto.Enum1_ENUM_1_VAL_1},
			},
			want:    &testproto.WithAllFieldTypes{FieldInt64: 418, Enum1: testproto.Enum1_ENUM_1_VAL_1},
			wantErr: false,
		},
		{
			name: "plain",
			args: args{
				message: &testproto.WithAllFieldTypes{FieldInt64: 418, FieldStringSensitive: "pad"},
			},
			want: &testproto.WithAllFieldTypes{FieldInt64: 418}, wantErr: false,
		},
		{
			name: "oneof",
			args: args{
				message: &testproto.WithAllFieldTypes{PaymentToken: &testproto.WithAllFieldTypes_Token{Token: "earnest"}},
			},
			want:    &testproto.WithAllFieldTypes{PaymentToken: &testproto.WithAllFieldTypes_Token{Token: "earnest"}},
			wantErr: false,
		},
		{
			name: "oneof hide",
			args: args{
				message: &testproto.WithAllFieldTypes{PaymentToken: &testproto.WithAllFieldTypes_Cryptogram{Cryptogram: "earnest"}},
			},
			want:    &testproto.WithAllFieldTypes{},
			wantErr: false,
		},
		{
			name: "complex",
			args: args{
				message: &testproto.WithAllFieldTypes{
					MessageListSensitive: []*testproto.WithAllFieldTypes_Internal{{FieldInt64: 915}, {FieldInt64: 843}},
					MessageList: []*testproto.WithAllFieldTypes_Internal{
						{FieldInt64: 145, FieldStringSensitive: "progress"},
						{FieldInt64: 309},
						{SensitiveMap: map[string]*testproto.WithAllFieldTypes_Internal{"surround": {}}},
						{MapWithSensitiveKey: map[string]*testproto.WithAllFieldTypes_Internal{
							"detail":        {FieldInt64: 948},
							"hide_this_key": {FieldInt64: 999},
						}},
						{MapWithSensitiveKeyIntKey: map[int64]*testproto.WithAllFieldTypes_Internal{87654: {FieldInt64: 948}, 642: {FieldInt64: 651}}},
						{},
						{
							Recursive: &testproto.WithAllFieldTypes_Internal{
								Recursive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           530,
									FieldStringSensitive: "improve",
									FieldIntSensitive:    434,
								},
								RecursiveSensitive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           732,
									FieldStringSensitive: "walk",
								},
							},
							RecursiveSensitive: &testproto.WithAllFieldTypes_Internal{
								Recursive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           813,
									FieldStringSensitive: "pardon",
								},
								RecursiveSensitive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           632,
									FieldStringSensitive: "wild",
								},
							},
						},
					},
				},
			},
			want: &testproto.WithAllFieldTypes{
				MessageList: []*testproto.WithAllFieldTypes_Internal{
					{FieldInt64: 145},
					{FieldInt64: 309},
					{},
					{MapWithSensitiveKey: map[string]*testproto.WithAllFieldTypes_Internal{
						"detail":        {FieldInt64: 948},
						"hide_this_key": {},
					}},
					{MapWithSensitiveKeyIntKey: map[int64]*testproto.WithAllFieldTypes_Internal{87654: {}, 642: {FieldInt64: 651}}},
					{},
					{
						Recursive: &testproto.WithAllFieldTypes_Internal{
							Recursive: &testproto.WithAllFieldTypes_Internal{
								FieldInt64: 530,
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := redactor.Redact(tt.args.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("SensetiveFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.True(t, proto.Equal(tt.want, tt.args.message))
			assert.Equal(t, string(must(json.Marshal(tt.want))), string(must(json.Marshal(tt.args.message))))
		})
	}
}
func TestRedactProto_SetStringClearOther(t *testing.T) {
	t.Parallel()
	redactor := Redactor{
		RedactingHandler: func(parent protoreflect.Value, field protoreflect.FieldDescriptor) error {
			if parent.IsValid() && field.Kind() == protoreflect.StringKind {
				parent.Message().Set(field, protoreflect.ValueOfString("REDACTED"))
			} else {
				parent.Message().Clear(field)
			}
			return nil
		},
		SensitiveFieldAnnotation: testproto.E_SensitiveData,
	}
	type args struct {
		message proto.Message
	}
	tests := []struct {
		name    string
		args    args
		want    proto.Message
		wantErr bool
	}{
		{
			name: "complex",
			args: args{
				message: &testproto.WithAllFieldTypes{
					MessageListSensitive: []*testproto.WithAllFieldTypes_Internal{{FieldInt64: 915}, {FieldInt64: 843}},
					MessageList: []*testproto.WithAllFieldTypes_Internal{
						{FieldInt64: 145, FieldStringSensitive: "progress"},
						{FieldInt64: 309},
						{SensitiveMap: map[string]*testproto.WithAllFieldTypes_Internal{"surround": {}}},
						{MapWithSensitiveKey: map[string]*testproto.WithAllFieldTypes_Internal{
							"detail":        {FieldInt64: 948, FieldStringSensitive: "Conubiafeugiat"},
							"hide_this_key": {FieldInt64: 999},
						}},
						{MapWithSensitiveKeyIntKey: map[int64]*testproto.WithAllFieldTypes_Internal{87654: {FieldInt64: 948}, 642: {FieldInt64: 651}}},
						{},
						{
							Recursive: &testproto.WithAllFieldTypes_Internal{
								Recursive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           530,
									FieldStringSensitive: "improve",
									FieldIntSensitive:    434,
								},
								RecursiveSensitive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           732,
									FieldStringSensitive: "walk",
								},
							},
							RecursiveSensitive: &testproto.WithAllFieldTypes_Internal{
								Recursive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           813,
									FieldStringSensitive: "pardon",
								},
								RecursiveSensitive: &testproto.WithAllFieldTypes_Internal{
									FieldInt64:           632,
									FieldStringSensitive: "wild",
								},
							},
						},
					},
				},
			},
			want: &testproto.WithAllFieldTypes{
				MessageList: []*testproto.WithAllFieldTypes_Internal{
					{FieldInt64: 145, FieldStringSensitive: "REDACTED"},
					{FieldInt64: 309},
					{},
					{MapWithSensitiveKey: map[string]*testproto.WithAllFieldTypes_Internal{
						"detail":        {FieldInt64: 948, FieldStringSensitive: "REDACTED"},
						"hide_this_key": {},
					}},
					{MapWithSensitiveKeyIntKey: map[int64]*testproto.WithAllFieldTypes_Internal{87654: {}, 642: {FieldInt64: 651}}},
					{},
					{
						Recursive: &testproto.WithAllFieldTypes_Internal{
							Recursive: &testproto.WithAllFieldTypes_Internal{
								FieldInt64:           530,
								FieldStringSensitive: "REDACTED",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := redactor.Redact(tt.args.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("SensetiveFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.True(t, proto.Equal(tt.want, tt.args.message))
			assert.Equal(t, string(must(json.Marshal(tt.want))), string(must(json.Marshal(tt.args.message))))
		})
	}
}

/*
goos: darwin
goarch: arm64
pkg: gitlab.int.tsum.com/preowned/simona/delta/core.git/dproto
Benchmark
Benchmark-10    	  217131	      5317 ns/op
*/
func Benchmark(b *testing.B) {
	msg := &testproto.WithAllFieldTypes{
		MessageList: []*testproto.WithAllFieldTypes_Internal{
			{FieldInt64: 145, FieldStringSensitive: "progress"},
			{FieldInt64: 309},
			{SensitiveMap: map[string]*testproto.WithAllFieldTypes_Internal{"surround": {}}},
			{MapWithSensitiveKey: map[string]*testproto.WithAllFieldTypes_Internal{
				"detail":        {FieldInt64: 948},
				"hide_this_key": {FieldInt64: 999},
			}},
			{MapWithSensitiveKeyIntKey: map[int64]*testproto.WithAllFieldTypes_Internal{87654: {FieldInt64: 948}, 642: {FieldInt64: 651}}},
			{},
		},
	}
	for i := 0; i < b.N; i++ {
		_ = Redact(msg, testproto.E_SensitiveData)
	}
}

func must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
