package supercall_test

import (
	"testing"

	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
)

func TestNew(t *testing.T) {
	s := supercall.New("https://ticket-api.superceed.com:9000/socket.io/?EIO=4&transport=polling", "TxbA20O4S0KO")
	err := s.SetupConnection()
	if err != nil {
		t.Fatal(err)
	}

	err = s.SendData(supercall.AgiEvent, supercall.AgiData{
		Source:      "source",
		AgiId:       "agiId",
		ExtensionId: "1234",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("success")
}
