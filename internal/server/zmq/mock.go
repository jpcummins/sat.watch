package zmq

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/bnkamalesh/errors"
	zmq "github.com/go-zeromq/zmq4"
	"github.com/rs/zerolog/log"
)

type MockZmqServer struct {
	txStream chan string
}

func Init(host string, port int) (*MockZmqServer, error) {
	logger := log.With().Str("module", "mockzmq").Str("host", host).Int("port", port).Logger()

	server := MockZmqServer{}
	server.txStream = make(chan string)

	go func(txStream <-chan string) {
		pub := zmq.NewPub(context.Background())
		defer pub.Close()

		err := pub.Listen("tcp://" + host + ":" + fmt.Sprint(port))
		if err != nil {
			logger.Fatal().Err(err).Msg("could not connect")
		}

		for {
			select {
			case tx := <-txStream:
				data, err := hex.DecodeString(tx)
				if err != nil {
					log.Print(errors.InternalErrf(err, "Unable to decode tx: '%s'", tx))
					continue
				}
				msg := zmq.NewMsgFrom(
					[]byte("rawtx"),
					data,
				)

				log.Debug().Str("component", "mockzmq").Msg("sent mock tx")
				err = pub.Send(msg)
				if err != nil {
					logger.Fatal().Err(err).Msg("could not send msg")
				}
			}
		}
	}(server.txStream)
	return &server, nil
}

func (mock MockZmqServer) SendTx(tx string) {
	mock.txStream <- tx
}
