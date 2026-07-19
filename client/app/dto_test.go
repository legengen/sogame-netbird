package app

import (
	"reflect"
	"strings"
	"testing"
)

func TestWailsDTOsContainNoEnrollmentSecrets(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(StateSnapshot{}),
		reflect.TypeOf(PeerSnapshot{}),
		reflect.TypeOf(ServiceSnapshot{}),
		reflect.TypeOf(CreateRoomRequest{}),
		reflect.TypeOf(JoinRoomRequest{}),
		reflect.TypeOf(SwitchRoomRequest{}),
		reflect.TypeOf(DiagnosticResult{}),
		reflect.TypeOf(PublicError{}),
	}

	for _, typ := range types {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			joined := strings.ToLower(field.Name + " " + field.Tag.Get("json"))
			for _, forbidden := range []string{"setupkey", "setup_key", "authorization", "admintoken", "privatekey"} {
				if strings.Contains(joined, forbidden) {
					t.Fatalf("%s.%s exposes forbidden field %q", typ.Name(), field.Name, forbidden)
				}
			}
		}
	}
}
