package worker

import (
	"context"
	"io"
	"time"

	"github.com/superfeelapi/goEagiSupercall/foundation/external/goVad"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
	pb "github.com/superfeelapi/goVad/proto"
)

func (w *Worker) goVadOperation() {
	w.logger.Infow("worker: goVadOperation: G started")
	defer w.logger.Infow("worker: goVadOperation: G completed")

	defer w.state.Set(state.GoVad, false)

	sessionID := <-w.idCh

	grpc := goVad.New(w.config.GrpcAddress, w.config.GrpcCertFilePath, w.config.Actor, w.config.AgiID, sessionID)
	err := grpc.SetupConnection()
	if err != nil {
		w.logger.Errorw("worker: goVadOperation", "ERROR", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = grpc.RegisterRoom(ctx)
	if err != nil {
		w.logger.Errorw("worker: goVadOperation", "ERROR", err)
		return
	}

	err = grpc.CheckRoomStatus(ctx)
	if err != nil {
		w.logger.Errorw("worker: goVadOperation", "ERROR", err)
		return
	}

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	stream, err := grpc.Client.Send(ctx)
	if err != nil {
		w.logger.Errorw("worker: goVadOperation", "ERROR", err)
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			default:
				_, err := stream.Recv()
				if err == io.EOF {
					w.logger.Errorw("worker: goVadOperation", "ERROR", err)
					cancel()
					return
				}
				if err != nil {
					w.logger.Errorw("worker: goVadOperation", "ERROR", err)
					cancel()
					return
				}
			}
		}
	}()

	w.logger.Infow("worker: goVadOperation: G listening")
	for {
		select {
		case vad := <-w.grpcCh:
			err := stream.Send(&pb.Data{
				Source:   w.config.Actor,
				AgiId:    w.config.AgiID,
				Detected: vad,
			})
			if err != nil {
				w.logger.Errorw("worker: goVadOperation", "ERROR", err)
				cancel()
				return
			}

		case <-ctx.Done():
			w.logger.Errorw("worker: goVadOperation", "ERROR", ctx.Err())
			return

		case <-w.shut:
			w.logger.Infow("worker: goVadOperation: received shut signal")
			cancel()
			return
		}
	}
}
