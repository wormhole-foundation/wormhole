package comm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/internal/testutils"
	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const workingServerSock = "127.0.0.1:5933"
const workingServerName = "127.0.0.1"
const workingServerPort = 5933

var workingServerAsMessageRecipient = []*tss.Identity{&tss.Identity{
	Hostname: workingServerName,
	Port:     workingServerPort,
}}

type mockTssMessageHandler struct {
	chn              chan tss.Sendable
	selfCert         *tls.Certificate
	peersToConnectTo []*x509.Certificate
	peerId           *tss.Identity
}

func (m *mockTssMessageHandler) GetCertificate() *tls.Certificate { return m.selfCert }
func (m *mockTssMessageHandler) GetPeers() []*x509.Certificate    { return m.peersToConnectTo }
func (m *mockTssMessageHandler) FetchPartyId(*x509.Certificate) (*tss.Identity, error) {
	return m.peerId, nil
}
func (m *mockTssMessageHandler) ProducedOutputMessages() <-chan tss.Sendable {
	return m.chn
}
func (m *mockTssMessageHandler) HandleIncomingTssMessage(msg tss.Incoming) {
	fmt.Println("received message from", msg.GetSource())
}

// wraps regular server and changes its Send function.
type testServer struct {
	*server
	atomic.Uint32
	done                         chan struct{}
	numberOfReconnectionAttempts int
	// when set to true, the server will block for 30 seconds.
	isMaliciousBlocker bool
}

func (w *testServer) Send(in tsscommv1.DirectLink_SendServer) error {
	prevVal := w.Uint32.Add(1)
	if int(prevVal) == w.numberOfReconnectionAttempts {
		close(w.done)
	}
	if w.isMaliciousBlocker {
		time.Sleep(time.Second * 30)
	}

	return io.EOF
}

func TestTLSConnectAndRedial(t *testing.T) {
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ctx = testutils.MakeSupervisorContext(ctx)

	en, err := _loadGuardians(2)
	a.NoError(err)

	tmpSrvr, err := NewServer(workingServerSock, supervisor.Logger(ctx), &mockTssMessageHandler{
		chn:      nil,
		selfCert: en[0].GetCertificate(),
		// connect to no one.
		peersToConnectTo: en[0].GetPeers(), // Give the peer a certificate.
		peerId:           &tss.Identity{},
	})
	a.NoError(err)

	tstServer := testServer{
		server:                       tmpSrvr.(*server),
		Uint32:                       atomic.Uint32{},
		done:                         make(chan struct{}),
		numberOfReconnectionAttempts: 2,
	}
	tstServer.server.ctx = ctx

	listener, err := net.Listen("tcp", workingServerSock)
	a.NoError(err)
	defer listener.Close()

	gserver := grpc.NewServer(tstServer.makeServerCredentials())
	defer gserver.Stop()

	tsscommv1.RegisterDirectLinkServer(gserver, &tstServer)
	go gserver.Serve(listener)

	PEMCert := en[0].GuardianStorage.TlsX509
	serverCert, err := internal.PemToCert(PEMCert)
	a.NoError(err)

	msgChan := make(chan tss.Sendable)
	srvr, err := NewServer("localhost:5930", supervisor.Logger(ctx), &mockTssMessageHandler{
		chn:              msgChan,
		selfCert:         en[1].GetCertificate(),
		peersToConnectTo: []*x509.Certificate{serverCert}, // will ask to fetch each peer (and return the below peerId)
		peerId: &tss.Identity{
			Cert:     serverCert,
			Hostname: workingServerName,
			Port:     workingServerPort,
		},
	})
	a.NoError(err)

	srv := srvr.(*server)
	srv.ctx = ctx
	// setting up server dailer and sender
	srv.run()
	time.Sleep(time.Second)

	//should cause disconnect
	msgChan <- &tss.Echo{
		Echo:       &tsscommv1.Echo{},
		Recipients: workingServerAsMessageRecipient,
	}
	time.Sleep(time.Second * 2)

	msgChan <- &tss.Unicast{
		Receipients: workingServerAsMessageRecipient,
	}

	select {
	case <-ctx.Done():
		t.FailNow()
	case <-tstServer.done:
	}
}

func TestRelentlessReconnections(t *testing.T) {
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	ctx = testutils.MakeSupervisorContext(ctx)

	en, err := _loadGuardians(2)
	a.NoError(err)

	PEMCert := en[0].GuardianStorage.TlsX509
	serverCert, err := internal.PemToCert(PEMCert)
	a.NoError(err)

	msgChan := make(chan tss.Sendable)
	srvr, err := NewServer("localhost:5930", supervisor.Logger(ctx), &mockTssMessageHandler{
		chn:              msgChan,
		selfCert:         en[1].GetCertificate(),
		peersToConnectTo: []*x509.Certificate{serverCert}, // will ask to fetch each peer (and return the below peerId)
		peerId: &tss.Identity{
			Cert:     serverCert,
			Hostname: workingServerName,
			Port:     workingServerPort,
		},
	})
	a.NoError(err)

	srv := srvr.(*server)
	srv.ctx = ctx
	// setting up server dailer and sender
	srv.run()

	tmpSrvr, err := NewServer(workingServerSock, supervisor.Logger(ctx), &mockTssMessageHandler{
		chn:      nil,
		selfCert: en[0].GetCertificate(),
		// connect to no one.
		peersToConnectTo: en[0].GetPeers(), // Give the peer a certificate.
		peerId:           &tss.Identity{},
	})
	a.NoError(err)

	tstServer := testServer{
		server:                       tmpSrvr.(*server),
		Uint32:                       atomic.Uint32{},
		done:                         make(chan struct{}),
		numberOfReconnectionAttempts: 5,
	}
	tstServer.server.ctx = ctx

	listener, err := net.Listen("tcp", workingServerSock)
	a.NoError(err)
	defer listener.Close()

	gserver := grpc.NewServer(tstServer.makeServerCredentials())
	defer gserver.Stop()

	tsscommv1.RegisterDirectLinkServer(gserver, &tstServer)
	go gserver.Serve(listener)

	for i := 0; i < 10; i++ {
		msgChan <- &tss.Unicast{
			Unicast: &tsscommv1.Unicast{
				Content: &tsscommv1.Unicast_Tss{
					Tss: &tsscommv1.TssContent{
						Payload:         []byte{1},
						MsgSerialNumber: 2,
					},
				},
			},
			Receipients: workingServerAsMessageRecipient,
		}

		select {
		case <-ctx.Done():
			t.FailNow()
		case <-tstServer.done:
			return // only way to pass the test.
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}

	t.FailNow()
}

type tssMockJustForMessageGeneration struct {
	tss.ReliableMessenger
	chn chan tss.Sendable
}

func (m *tssMockJustForMessageGeneration) ProducedOutputMessages() <-chan tss.Sendable {
	return m.chn
}
func TestNonBlockedBroadcast(t *testing.T) {
	a := require.New(t)

	workingServers := []string{"localhost:5500", "localhost:5501"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()
	ctx = testutils.MakeSupervisorContext(ctx)

	en, err := _loadGuardians(3)
	a.NoError(err)

	donechns := make([]chan struct{}, 2)
	// set servers up.
	for i := 0; i < 2; i++ {
		tmpSrvr, err := NewServer(workingServers[i], supervisor.Logger(ctx), &mockTssMessageHandler{
			chn:              nil,
			selfCert:         en[i].GetCertificate(),
			peersToConnectTo: en[0].GetPeers(), // Give the peer a certificate.
			peerId:           &tss.Identity{},
		})
		a.NoError(err)

		tstServer := testServer{
			server:                       tmpSrvr.(*server),
			Uint32:                       atomic.Uint32{},
			done:                         make(chan struct{}),
			numberOfReconnectionAttempts: 1,
			isMaliciousBlocker:           true,
		}
		donechns[i] = tstServer.done
		tstServer.server.ctx = ctx

		listener, err := net.Listen("tcp", workingServers[i])
		a.NoError(err)
		defer listener.Close()

		gserver := grpc.NewServer(tstServer.makeServerCredentials())
		defer gserver.Stop()

		tsscommv1.RegisterDirectLinkServer(gserver, &tstServer)
		go gserver.Serve(listener)
	}

	for _, v := range en[2].GuardianStorage.Guardians.Identities {
		v.Hostname = "localhost"
		if bytes.Equal(v.KeyPEM, en[0].Self.KeyPEM) {
			v.Port = 5500

			continue
		}

		if bytes.Equal(v.KeyPEM, en[1].Self.KeyPEM) {
			v.Port = 5501

			continue
		}

		v.Hostname = ""
	}

	msgChan := make(chan tss.Sendable)
	srvr, err := NewServer("localhost:5930", supervisor.Logger(ctx), &tssMockJustForMessageGeneration{
		ReliableMessenger: en[2],
		chn:               msgChan,
	})
	a.NoError(err)

	srv := srvr.(*server)
	srv.ctx = ctx
	// setting up server dailer and sender
	srv.run()
	time.Sleep(time.Second)

	numDones := 0
	for i := 0; i < 10; i++ {
		msgChan <- &tss.Echo{
			Recipients: workingServerAsMessageRecipient,
		}

		select {
		case <-ctx.Done():
			t.FailNow()
		case <-donechns[0]:
			numDones += 1
		case <-donechns[1]:
			numDones += 1
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
	if numDones >= 2 {
		return
	}

	cancel()
	t.FailNow()

}

func TestBackoff(t *testing.T) {
	a := require.New(t)
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*60)
	defer cncl()

	t.Run("basic1", func(t *testing.T) {
		heap := newBackoffHeap()

		heap.Enqueue("3")
		a.Equal("3", heap.Dequeue())
		heap.Enqueue("3")
		heap.Enqueue("1")
		heap.Enqueue("2")

		expected := []string{"1", "2", "3"}
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				t.FailNow()
			case <-heap.WaitOnTimer():
				hostname := heap.Dequeue()
				a.Equal(expected[i], hostname)
			}
		}
	})

	t.Run("basic2", func(t *testing.T) {
		heap := newBackoffHeap()

		heap.Enqueue("1")
		a.Equal("1", heap.Dequeue())
		heap.ResetAttempts("1")
		heap.Enqueue("1")
		heap.Enqueue("2")
		heap.Enqueue("3")

		expected := []string{"1", "2", "3"}
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				t.FailNow()
			case <-heap.WaitOnTimer():
				hostname := heap.Dequeue()
				a.Equal(expected[i], hostname)
			}
		}
	})

	t.Run("complex", func(t *testing.T) {
		heap := newBackoffHeap()

		// operations on an empty heap:
		a.Equal("", heap.Dequeue())

		heap.ResetAttempts("1")
		heap.Enqueue("1")
		heap.Enqueue("1")
		heap.Enqueue("1")
		heap.Enqueue("1")
		a.Equal("1", heap.Dequeue())

		heap.ResetAttempts("1")
		heap.Enqueue("1")
		heap.Enqueue("2")
		heap.Enqueue("3")
		heap.Enqueue("2")
		a.Equal("1", heap.Dequeue())
		a.Equal("2", heap.Dequeue())

		heap.ResetAttempts("2")
		heap.Enqueue("2")
		heap.Enqueue("4")
		heap.Enqueue("5")

		expected := []string{"3", "2", "4", "5"}
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				t.FailNow()
			case <-heap.WaitOnTimer():
				hostname := heap.Dequeue()
				a.Equal(expected[i], hostname)
			}
		}
	})

	t.Run("maxAndMinValue", func(t *testing.T) {
		maxBackoffTime := (&dialWithBackoff{attempt: maxBackoffTimeModifier})._durationBasedOnNumberOfAttempts()

		heap := newBackoffHeap()

		heap.attemptsPerPeer["1"] = 23144532345345665 // large number.
		heap.Enqueue("1")
		v := heap.Peek()

		a.True(v.nextRedialTime.Before(time.Now().Add(maxBackoffTime)))
		a.True(v.nextRedialTime.After(time.Now().Add(maxBackoffTime - time.Second)))

		a.Equal("1", heap.Dequeue())
		timenow := time.Now()
		heap.ResetAttempts("1")
		heap.Enqueue("1")
		v = heap.Peek()
		a.True(v.nextRedialTime.Before(timenow.Add(minBackoffTime + 10*time.Millisecond)))
		a.True(v.nextRedialTime.After(timenow.Add(minBackoffTime - 10*time.Millisecond)))

	})
}

// TODO: this is a copy-paste from tss/implementation_test.go
func loadMockGuardianStorage(gstorageIndex int) (*tss.GuardianStorage, error) {
	path, err := testutils.GetMockGuardianTssStorage(gstorageIndex)
	if err != nil {
		return nil, err
	}

	st, err := tss.NewGuardianStorageFromFile(path)
	if err != nil {
		return nil, err
	}
	return st, nil
}

// TODO: this is a copy-paste from tss/implementation_test.go
func _loadGuardians(numParticipants int) ([]*tss.Engine, error) {
	engines := make([]*tss.Engine, numParticipants)

	for i := 0; i < numParticipants; i++ {
		gs, err := loadMockGuardianStorage(i)
		if err != nil {
			return nil, err
		}

		e, err := tss.NewReliableTSS(gs)
		if err != nil {
			return nil, err
		}

		en, ok := e.(*tss.Engine)
		if !ok {
			return nil, errors.New("not an engine")
		}
		engines[i] = en
	}

	return engines, nil
}

type testCAInspectionFailForNonCACerts struct {
	*server
	atomic.Uint32
	done                         chan struct{}
	numberOfReconnectionAttempts int
	// when set to true, the server will block for 30 seconds.
	isMaliciousBlocker bool
}

func TestNotAcceptNonCAs(t *testing.T) {
	a := require.New(t)

	en, err := _loadGuardians(2)
	a.NoError(err)

	// ============
	// Creating new Cert which is NOT a CA
	// ============

	serverCert, err := internal.PemToCert(en[0].GuardianStorage.TlsX509)
	a.NoError(err)

	rootKey, err := internal.PemToPrivateKey(en[0].PrivateKey)
	a.NoError(err)
	clientTlsCert, clientCert := tlsCert(serverCert, rootKey)

	// ============
	// setting server up, with this Cert allowed
	// ============

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	ctx = testutils.MakeSupervisorContext(ctx)

	tmp, err := NewServer(workingServerSock, supervisor.Logger(ctx), &mockTssMessageHandler{
		chn:      nil,
		selfCert: en[0].GetCertificate(),
		// connect to no one.
		peersToConnectTo: []*x509.Certificate{clientCert}, // Give the peer a certificate.
		peerId:           &tss.Identity{},
	})
	a.NoError(err)

	server := tmp.(*server)
	server.ctx = ctx

	listener, err := net.Listen("tcp", workingServerSock)
	a.NoError(err)
	defer listener.Close()

	gserver := grpc.NewServer(server.makeServerCredentials())
	defer gserver.Stop()

	tsscommv1.RegisterDirectLinkServer(gserver, server)
	go gserver.Serve(listener)

	time.Sleep(time.Millisecond * 200)
	// ============
	// trying to send message using cert
	// ============
	pool := x509.NewCertPool()

	runningServerX509, err := internal.PemToCert(en[0].GuardianStorage.TlsX509)
	a.NoError(err)

	pool.AddCert(runningServerX509) // dialing to peer and accepting his cert only.

	cc, err := grpc.Dial(workingServerSock,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion:   tls.VersionTLS13,                  // tls 1.3
			Certificates: []tls.Certificate{*clientTlsCert}, // our cert to be sent to the peer.
			RootCAs:      pool,
		})),
	)
	a.NoError(err)

	defer cc.Close()

	stream, err := tsscommv1.NewDirectLinkClient(cc).Send(ctx)
	a.NoError(err)

	stream.Send(&tsscommv1.PropagatedMessage{})
	_, err = stream.CloseAndRecv()
	a.ErrorContains(err, "not a CA")
}

func tlsCert(rootCA *x509.Certificate, rootKey *ecdsa.PrivateKey) (*tls.Certificate, *x509.Certificate) {
	template := *rootCA
	// this cert will be the CA that we will use to sign the server cert
	template.IsCA = false
	// describe what the certificate will be used for
	template.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	pubcert, certpem, err := internal.CreateCert(&template, rootCA, &priv.PublicKey, rootKey)
	if err != nil {
		panic(err)
	}

	tlscert, err := tls.X509KeyPair(certpem, internal.PrivateKeyToPem(priv))
	if err != nil {
		panic(err)
	}
	return &tlscert, pubcert
}

func TestDialWithDefaultPort(t *testing.T) {
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()
	ctx = testutils.MakeSupervisorContext(ctx)

	en, err := _loadGuardians(3)
	a.NoError(err)

	communicatingEngine := en[0]
	listenerEngine := en[1]

	listenerServerPath := "localhost:" + tss.DefaultPort
	// set up server that only listent and aren't able to connect to anyone.
	listenerServer, err := NewServer(listenerServerPath, supervisor.Logger(ctx), &mockTssMessageHandler{
		chn:      nil,
		selfCert: listenerEngine.GetCertificate(),
		// the listening server will expect this cert to connect with.
		peersToConnectTo: []*x509.Certificate{communicatingEngine.GetCertificate().Leaf},
		peerId:           &tss.Identity{},
	})
	a.NoError(err)

	ListenerWrapper := testServer{
		server:                       listenerServer.(*server),
		Uint32:                       atomic.Uint32{},
		done:                         make(chan struct{}),
		numberOfReconnectionAttempts: 1,
		isMaliciousBlocker:           true,
	}

	ListenerWrapper.server.ctx = ctx

	l, err := net.Listen("tcp", listenerServerPath)
	a.NoError(err)
	defer l.Close()

	gserver := grpc.NewServer(ListenerWrapper.makeServerCredentials())
	defer gserver.Stop()

	tsscommv1.RegisterDirectLinkServer(gserver, &ListenerWrapper)
	go gserver.Serve(l)

	//  SETTING THE ID TO CONNECT TO WITHOUT A PORT:
	// ensuring the communicating server will have to use the default port to dial.
	for _, v := range communicatingEngine.Guardians.Identities {
		if bytes.Equal(v.KeyPEM, listenerEngine.Self.KeyPEM) {
			v.Hostname = "localhost"

			continue
		}

		v.Hostname = ""
	}

	msgChan := make(chan tss.Sendable)
	communicator, err := NewServer("localhost:5930", supervisor.Logger(ctx), &tssMockJustForMessageGeneration{
		ReliableMessenger: communicatingEngine,
		chn:               msgChan,
	})
	a.NoError(err)

	tmp := communicator.(*server)
	tmp.ctx = ctx
	tmp.run()

	time.Sleep(time.Second)

	for i := 0; i < 10; i++ {
		msgChan <- &tss.Echo{
			Recipients: workingServerAsMessageRecipient,
		}

		select {
		case <-ctx.Done():
			t.FailNow()
		case <-ListenerWrapper.done:
			return
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}

	t.FailNow()
}

type mockJustHandleIncomingMessage struct {
	tss.ReliableMessenger
	receivedData chan tss.Incoming
}

func (m *mockJustHandleIncomingMessage) HandleIncomingTssMessage(msg tss.Incoming) {
	m.receivedData <- msg
	close(m.receivedData)
}

func TestDialWithDefaultPortDeliverCorrectSrc(t *testing.T) {
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()
	ctx = testutils.MakeSupervisorContext(ctx)

	en, err := _loadGuardians(3)
	a.NoError(err)

	streamReceiverEngine := en[0]
	senderEngine := en[1]

	streamReceiverPath := "localhost"

	// ensuring when a message arrives, the server idetntifies the source according to
	// the tls key, then returns the tss.PartyID according to what is
	// stored in the guardian storage.
	expectedText := "This text is what i expect to see in the incoming message."
	for _, v := range streamReceiverEngine.Guardians.Identities {
		if bytes.Equal(v.KeyPEM, senderEngine.Self.KeyPEM) {
			v.Hostname = expectedText

			continue
		}

		v.Pid.Id = ""
	}

	// Setting the ID as they will be sent and used to connect to the other party.
	for _, v := range senderEngine.Guardians.Identities {
		if bytes.Equal(v.KeyPEM, streamReceiverEngine.Self.KeyPEM) {
			v.Hostname = streamReceiverPath // ensuring the server connects
			v.Port = 5930
			continue
		}

		v.Hostname = ""
	}

	incomingDataChan := make(chan tss.Incoming)
	listenerServer, err := NewServer(streamReceiverPath, supervisor.Logger(ctx),
		&mockJustHandleIncomingMessage{
			ReliableMessenger: streamReceiverEngine,
			receivedData:      incomingDataChan,
		},
	)
	a.NoError(err)

	ListenerWrapper := listenerServer.(*server)
	ListenerWrapper.ctx = ctx

	l, err := net.Listen("tcp", streamReceiverPath+":5930")
	a.NoError(err)
	defer l.Close()

	gserver := grpc.NewServer(ListenerWrapper.makeServerCredentials())
	defer gserver.Stop()

	tsscommv1.RegisterDirectLinkServer(gserver, ListenerWrapper)
	go gserver.Serve(l)

	msgChan := make(chan tss.Sendable)
	sender, err := NewServer("nonsensePort", supervisor.Logger(ctx), &tssMockJustForMessageGeneration{
		ReliableMessenger: senderEngine,
		chn:               msgChan,
	})
	a.NoError(err)

	tmp := sender.(*server)
	tmp.ctx = ctx
	tmp.run() // demanding this server run.

	time.Sleep(time.Second * 1)

	//should set up connection with the stream r

	msgChan <- &tss.Echo{
		Echo: &tsscommv1.Echo{},
		Recipients: []*tss.Identity{
			{
				CommunicationIndex: 0,
				Hostname:           streamReceiverPath,
				Port:               5930,
			},
		},
	}

	select {
	case <-ctx.Done():
		t.FailNow()
	case incoming := <-incomingDataChan:
		// ensuring the incoming message has the hostname i've eddited (proof we are matching the TLS cert,
		//  with the correct identity, even if things like hostname change)
		a.Equal(expectedText, incoming.GetSource().Hostname)
		return
	}
}

// uses the reliable messenger to send messages, but mocks the ProducedOutputMessages
type mockProduceOutputMessages struct {
	mockJustHandleIncomingMessage
	fakeDataToSendChan chan tss.Sendable
}

func (m *mockProduceOutputMessages) ProducedOutputMessages() <-chan tss.Sendable {
	return m.fakeDataToSendChan
}

func TestConnectingToServers(t *testing.T) {
	t.SkipNow() // Manual test, help inspect connections via logs.
	a := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()
	ctx = testutils.MakeSupervisorContext(ctx)

	en, err := _loadGuardians(5)
	a.NoError(err)
	// ============
	incomingMsgChn := make([]chan tss.Incoming, len(en))
	channelForGeneratedMessages := make(chan tss.Sendable)
	// setting up servers
	srvrs := make([]*server, 5)
	for i, e := range en {
		incomingMsgChn[i] = make(chan tss.Incoming)
		e.GuardianStorage.Self.Hostname = "localhost"
		e.GuardianStorage.Self.Port = 5930 + i
		for j, id := range e.Guardians.Identities {
			id.Hostname = "localhost"
			id.Port = 5930 + j
		}

		e.Start(ctx)
		s, err := NewServer(e.GuardianStorage.Self.NetworkName(), supervisor.Logger(ctx), &mockProduceOutputMessages{
			mockJustHandleIncomingMessage: mockJustHandleIncomingMessage{
				ReliableMessenger: e,
				receivedData:      incomingMsgChn[i],
			},

			fakeDataToSendChan: channelForGeneratedMessages,
		})
		a.NoError(err)

		srvrs[i] = s.(*server)

	}

	for _, s := range srvrs {
		go s.Run(ctx)
	}

	// ============
	time.Sleep(time.Second * 2)
	channelForGeneratedMessages <- &tss.Echo{
		Recipients: en[0].Guardians.Identities,
		Echo:       &tsscommv1.Echo{},
	}
	time.Sleep(time.Second * 2)

	for _, chn := range incomingMsgChn {
		select {
		case <-chn:
		case <-ctx.Done():
			t.FailNow()
		}
	}
	cancel()
	time.Sleep(100 * time.Millisecond)

	return
}
