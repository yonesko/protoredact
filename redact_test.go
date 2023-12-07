package protoredact

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/yonesko/protoredact/testproto/go/testproto"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestRedactProto(t *testing.T) {
	t.Parallel()
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
			err := Redact(tt.args.message, testproto.E_SensitiveData)
			if (err != nil) != tt.wantErr {
				t.Errorf("SensetiveFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.True(t, proto.Equal(tt.want, tt.args.message))
			assert.Equal(t, string(lo.Must(json.Marshal(tt.want))), string(lo.Must(json.Marshal(tt.args.message))))
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

func encrypt(plaintext string) string {
	aes, err := aes.NewCipher([]byte("N1PCdw3M2B1TfJhoaY2mL736p2vCUc47"))
	if err != nil {
		panic(err)
	}

	gcm := lo.Must(cipher.NewGCM(aes))
	nonce := make([]byte, gcm.NonceSize())
	//_, _ = io.ReadFull(rand.Reader, nonce)

	return string(gcm.Seal(nonce, nonce, []byte(plaintext), nil))
}

func Test_wfef(t *testing.T) {
	// This will successfully encrypt.

	// This will cause an error since the
	// plaintext is less than 16 bytes.
	ciphertext := encrypt("jhytgfl,ikmujnyhbtgrvfecdrgthyjukil,o.,hbgfvcdexwefrgvthyip8;nrytbevwfcdfr")
	fmt.Printf("Ciphertext: %s \n", base64.StdEncoding.EncodeToString([]byte(ciphertext)))

	s := decrypt(ciphertext)

	fmt.Printf("s='%+v'\n", s)
}

//000000000000000000000000eadc2924663fdbac41733228a465211d87a2ea1d126eb6bc1aaa94117503e4ba78c6842f18727f0bd0acff5ac6f905d95351

func decrypt(ciphertext string) string {
	aes, err := aes.NewCipher([]byte("N1PCdw3M2B1TfJhoaY2mL736p2vCUc47"))
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		panic(err)
	}

	// Since we know the ciphertext is actually nonce+ciphertext
	// And len(nonce) == NonceSize(). We can separate the two.
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		panic(err)
	}

	return string(plaintext)
}
