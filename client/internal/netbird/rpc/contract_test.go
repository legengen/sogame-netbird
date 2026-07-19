package proto

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestRequiredDaemonRPCContract(t *testing.T) {
	service := File_daemon_proto.Services().ByName("DaemonService")
	if service == nil {
		t.Fatal("DaemonService descriptor is missing")
	}
	requiredMethods := []string{
		"Login",
		"Up",
		"Status",
		"Down",
		"SubscribeEvents",
		"SwitchProfile",
		"AddProfile",
		"RemoveProfile",
		"ListProfiles",
		"GetActiveProfile",
		"Logout",
	}
	for _, name := range requiredMethods {
		if service.Methods().ByName(protoreflect.Name(name)) == nil {
			t.Errorf("required v0.74.7 daemon method %s is missing", name)
		}
	}

	criticalFields := map[string]map[string]int32{
		"LoginRequest": {
			"setupKey":      1,
			"managementUrl": 3,
			"profileName":   30,
			"username":      31,
		},
		"StatusResponse": {
			"status":        1,
			"fullStatus":    2,
			"daemonVersion": 3,
		},
		"PeerState": {
			"IP":         1,
			"connStatus": 3,
			"relayed":    5,
		},
	}
	for messageName, fields := range criticalFields {
		message := File_daemon_proto.Messages().ByName(protoreflect.Name(messageName))
		if message == nil {
			t.Fatalf("message %s is missing", messageName)
		}
		for fieldName, number := range fields {
			field := message.Fields().ByName(protoreflect.Name(fieldName))
			if field == nil || int32(field.Number()) != number {
				t.Errorf("%s.%s field number mismatch", messageName, fieldName)
			}
		}
	}
}
