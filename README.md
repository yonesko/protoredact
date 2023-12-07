Lib for fields redaction (hiding sensitive fields) base on filed options.

```bash
go get github.com/yonesko/protoredact
```

### Base case

Define field option:

```protobuf
syntax = "proto3";
package testproto;
import "google/protobuf/descriptor.proto";

message SensitiveData {
}

extend google.protobuf.FieldOptions {
  SensitiveData sensitive_data = 1200;
}
```

Annotate your message:

```protobuf
message WithAllFieldTypes {
  string fieldString = 1;
  string fieldStringSensitive = 2 [(sensitive_data) = {}];
}
```

Call Redact on your message:

```go
func Test(t *testing.T) {
msg := &testproto.WithAllFieldTypes{FieldInt64: 515, FieldStringSensitive: "my_password"}
msgCloned := proto.Clone(msg)
_ = Redact(msgCloned, testproto.E_SensitiveData)
bytesOriginal, _ := json.Marshal(msg)
bytesCloned, _ := json.Marshal(msgCloned)
fmt.Println("original:", string(bytesOriginal))
fmt.Println("cloned:", string(bytesCloned))
}
//original: {"fieldInt64":515, "fieldStringSensitive":"my_password"}
//cloned: {"fieldInt64":515}
```

Sensitive keys will be empty

### Map case

You can specify which keys of map to hide:

```protobuf
syntax = "proto3";
package testproto;
import "google/protobuf/descriptor.proto";

message SensitiveData {
  repeated string mapKeysToRedact = 1;
}

extend google.protobuf.FieldOptions {
  SensitiveData sensitive_data = 1200;
}

message WithAllFieldTypes {
  map<int64, string> mapWithSensitiveKeyIntKey = 1[(sensitive_data) = {mapKeysToRedact:["password"]}];
}
```

### Set Value Case

You can use `redactingHandler` to specify what you want to do with sensitive field
