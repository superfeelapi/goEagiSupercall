package goVad

import (
	"context"

	pb "github.com/superfeelapi/goVad/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Vad struct {
	Source            string
	AgiID             string
	SocketioSessionID string

	certPath    string
	grpcAddress string
	Client      pb.VadClient
}

func New(grpcAddress string, certPath string, source, agiID, socketSessionID string) *Vad {
	return &Vad{
		certPath:          certPath,
		grpcAddress:       grpcAddress,
		Source:            source,
		AgiID:             agiID,
		SocketioSessionID: socketSessionID,
	}
}

func (v *Vad) SetupConnection() error {
	creds, sslErr := credentials.NewClientTLSFromFile(v.certPath, "")
	if sslErr != nil {
		return sslErr
	}
	conn, err := grpc.Dial(v.grpcAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		return err
	}

	v.Client = pb.NewVadClient(conn)
	return nil
}

func (v *Vad) RegisterRoom(ctx context.Context) error {
	_, err := v.Client.Register(ctx, &pb.Room{
		Source:            v.Source,
		AgiId:             v.AgiID,
		SocketioSessionId: v.SocketioSessionID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (v *Vad) CheckRoomStatus(ctx context.Context) error {
	_, err := v.Client.CheckRoomStatus(ctx, &pb.Status{AgiId: v.AgiID})
	if err != nil {
		return err
	}

	return nil
}
