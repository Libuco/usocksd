package socks

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
	"bufio"
	"errors"
	"bytes"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/netutil"
	"github.com/cybozu-go/well"
)

const (
	copyBufferSize     = 64 << 10
	negotiationTimeout = 10 * time.Second
)

var (
	dialer = &net.Dialer{
		DualStack: true,
	}
)

// Authenticator is the interface for user authentication.
// It should look Username and Password field in the request and
// returns true if authentication succeeds.
//
// Note that both Username and Password may be empty.
type Authenticator interface {
	Authenticate(r *Request) bool
}

// RuleSet is the interface for access control.
// It should look the request properties and returns true
// if the request matches rules.
type RuleSet interface {
	Match(r *Request) bool
}

// Dialer is the interface to establish connection to the destination.
type Dialer interface {
	Dial(r *Request) (net.Conn, error)
}

// Server implement SOCKS protocol.
type Server struct {
	// Auth can be used to authenticate a request.
	// If nil, all requests are allowed.
	Auth Authenticator

	// Rules can be used to test a request if it matches rules.
	// If nil, all requests passes.
	Rules RuleSet

	// Dialer is used to make connections to destination servers.
	// If nil, net.DialContext is used.
	Dialer Dialer

	// Logger can be used to provide a custom logger.
	// If nil, the default logger is used.
	Logger *log.Logger

	// ShutdownTimeout is the maximum duration the server waits for
	// all connections to be closed before shutdown.
	//
	// Zero duration disables timeout.
	ShutdownTimeout time.Duration

	// Env is the environment where this server runs.
	//
	// The global environment is used if Env is nil.
	Env *well.Environment

	// SilenceLogs changes Info-level logs to Debug-level ones.
	SilenceLogs bool

	once   sync.Once
	server well.Server
	pool   *sync.Pool
}

func (s *Server) init() {
	if s.Logger == nil {
		s.Logger = log.DefaultLogger()
	}
	s.server.ShutdownTimeout = s.ShutdownTimeout
	s.server.Env = s.Env
	s.server.Handler = s.handleConnection
	s.pool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, copyBufferSize)
		},
	}
}

// Serve starts a goroutine to accept connections.
// This returns immediately.  l will be closed when s.Env is canceled.
// See https://godoc.org/github.com/cybozu-go/well#Server.Serve
func (s *Server) Serve(l net.Listener) {
	s.once.Do(s.init)
	s.server.Serve(l)
}

func (s *Server) dial(ctx context.Context, r *Request, network string) (net.Conn, error) {
	if s.Dialer != nil {
		return s.Dialer.Dial(r)
	}

	var addr string
	if len(r.Hostname) == 0 {
		addr = net.JoinHostPort(r.IP.String(), strconv.Itoa(r.Port))
	} else {
		addr = net.JoinHostPort(r.Hostname, strconv.Itoa(r.Port))
	}

	ctx, cancel := context.WithTimeout(ctx, negotiationTimeout)
	defer cancel()
	return dialer.DialContext(ctx, network, addr)
}

// handleConnection implements SOCKS protocol.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	conn.SetDeadline(time.Now().Add(negotiationTimeout))

	var preamble [2]byte
	_, err := io.ReadFull(conn, preamble[:])
	if err != nil {
		fields := well.FieldsFromContext(ctx)
		fields["client_addr"] = conn.RemoteAddr().String()
		fields[log.FnError] = err.Error()
		s.Logger.Error("failed to read preamble", fields)
		return
	}

	//initreader := bufio.NewReader(conn)
	/*buf,erro := io.ReadAll(conn)

	s.Logger.Error("buf_length:" + strconv.Itoa(len(buf)),  well.FieldsFromContext(ctx))

	if (erro != nil) {
		s.Logger.Error("buf_read_err:" + erro.Error(),  well.FieldsFromContext(ctx))
		conn.Close()
	}
	reader := bytes.NewReader(buf)
	*/
	

	//savedreader := bytes.NewReader(&buf)

	//ctx = context.WithValue(ctx,"reader",r)
	//ctx = context.WithValue(ctx,"buffer",buf)
	//ctx := context.WithValue(ctx,"reader2",r)

	

	i := 0
	handleConnectionSocks(s, ctx, conn, preamble, i)
	
}

func handleConnectionSocks(s *Server, ctx context.Context, conn net.Conn, preamble [2]byte, i int) {
	
	receiveFromClient := make(chan byte)
	sendToClient := make(chan byte)
	receiveFromServer := make(chan byte)
	sendToServer := make(chan byte)
	Use(receiveFromClient,sendToClient,receiveFromServer, sendToServer)

	//unfuckReceiveFromClient := make(chan bool)
	//unfuckSendToClient := make(chan bool)
	//unfuckReceiveFromServer := make(chan bool)
	//unfuckSendToServer := make(chan bool)

	//defer close(receiveFromClient)
	//defer  close(receiveFromServer)
	//defer  close(sendToClient)
	//defer  close(sendToServer)

	//defer  close(unfuckReceiveFromClient)
	//defer  close(unfuckSendToClient)
	//defer  close(unfuckReceiveFromServer)
	//defer  close(unfuckSendToServer)


	bufreader := bufio.NewReader(conn)
	bufwriter := bufio.NewWriter(conn)
	Use(bufwriter)
	
	var bufferRead bytes.Buffer
	reader := io.TeeReader(bufreader, &bufferRead)
	//var retry bytes.Buffer

	//var requestBufferRead bytes.Buffer

	for i < 100 {		
		var destConn net.Conn

		var bufferReadNew bytes.Buffer
		if i >=1 {
			s.Logger.Info("bufferRead is " + strconv.Itoa(bufferRead.Len()), well.FieldsFromContext(ctx))
			bufreader = bufio.NewReader(&bufferRead)
			reader = io.TeeReader(bufreader, &bufferReadNew)
		}
		
		s.Logger.Info("I:"+strconv.Itoa(i), well.FieldsFromContext(ctx))

		//reader.Seek(0,io.SeekStart)
		connVer := version(preamble[0])
		
		switch connVer {
		case SOCKS4:
			if i>=1 {
				destConn = s.handleSOCKS4(ctx, nil, preamble[1],i,reader)
			} else {
				destConn = s.handleSOCKS4(ctx, conn, preamble[1],i,reader)
			}
			if destConn == nil {
				s.Logger.Error("destConn is nil", well.FieldsFromContext(ctx))
				return
			}
		case SOCKS5:
			destConn = s.handleSOCKS5(ctx, conn, preamble[1])
			if destConn == nil {
				return
			}
		default:
			fields := well.FieldsFromContext(ctx)
			fields["client_addr"] = conn.RemoteAddr().String()
			s.Logger.Error("unknown SOCKS version", fields)
			return
		}

		//s.Logger.Info("buf is " + strconv.Itoa(bufferRead.Len()), well.FieldsFromContext(ctx))
		

		defer destConn.Close()
		netutil.SetKeepAlive(destConn)

		// negotiation completed.
		var zeroTime time.Time
		conn.SetDeadline(zeroTime)

		// do proxy
		
		//var requestBufferReadNew bytes.Buffer
		//nextbufreader := bufio.NewReader(conn)
		//nextTee := io.TeeReader(conn, &requestBufferRead)
		//if i > 0 {
		//	nextbufreader = bufio.NewReader(&requestBufferRead)
		//	nextTee = io.TeeReader(bufreader, &requestBufferReadNew)
		//}

		env, clientUnfuck, halfass := makeReadTeeChannel(s, ctx, conn, destConn, reader, bufreader, receiveFromClient)

		destReader := bufio.NewReader(destConn)
		env1, _ := makeReadChannel(s, ctx, conn, destConn, destReader,bufwriter,receiveFromServer, halfass)

		env.Stop()
		env1.Stop()
		err1 := env1.Wait()

		s.Logger.Info("Released server but not client:" + strconv.Itoa(i), well.FieldsFromContext(ctx))
		
		if (err1 != nil && "RETRY" == err1.Error()) {
			s.Logger.Info("Retrying due to signal:" + strconv.Itoa(i), well.FieldsFromContext(ctx))
			clientUnfuck <- true
		} else {
			//err := env.Wait()

			if (err1 != nil && "FIN" == err1.Error()) {
				clientUnfuck <- true
				env.Cancel(errors.New("CANCEL"))
				s.Logger.Info("Finished Successfully:" + strconv.Itoa(i), well.FieldsFromContext(ctx))
				return
			}

		}

		
		//if i > 0 {
		//	s.Logger.Info("Reassigning REQUESTbuffer with bytes:" + strconv.Itoa(requestBufferReadNew.Len()), well.FieldsFromContext(ctx))
		//	requestBufferRead = requestBufferReadNew
		//} else {
		//	s.Logger.Info("Request buffer filled with bytes:" + strconv.Itoa(requestBufferRead.Len()), well.FieldsFromContext(ctx))
		//}
		if i >= 1 {
			s.Logger.Info("Reassigning buffer with bytes:" + strconv.Itoa(bufferReadNew.Len()), well.FieldsFromContext(ctx))
			bufferRead = bufferReadNew
		}
		//env.Wait()
		//s.Logger.Info("Released client too:" + strconv.Itoa(i), well.FieldsFromContext(ctx))


		time.Sleep(50 * time.Millisecond)

		//envSendToClient,unfuckSendToClient := makeWriteChannel(s,ctx, conn, bufwriter,sendToClient)

		//destWriter := bufio.NewWriter(destConn)
		//envSendToServer,unfuckSendToServer := makeWriteChannel(s,ctx, destConn, destWriter,sendToServer)

		//yobaEnv, yobaUnfuck := yobaSync(s,ctx, receiveFromClient,receiveFromServer, sendToClient, sendToServer)

		//Use(envReceiveFromClient, unfuckReceiveFromClient, halfass, envReceiveFromServer,unfuckReceiveFromServer,envSendToClient, unfuckSendToClient, envSendToServer, unfuckSendToServer, yobaEnv, yobaUnfuck  )
		
		//envReceiveFromClient.Stop()
		//envReceiveFromServer.Stop()
		//envSendToClient.Stop()
		//envSendToServer.Stop()
		//yobaEnv.Stop()
		//yobaEnv.Wait()
		
		//handleConnectionPipe( s, ctx , conn, destConn, i, retry)

		//if retry.Len() == 0 {
		//	s.Logger.Info("Finished Successfully:" + strconv.Itoa(i), well.FieldsFromContext(ctx))
		//	return
		//}

		//s.Logger.Info("buf1 is " + strconv.Itoa(buf1.Len()), well.FieldsFromContext(ctx))

		
		i++
		//break
	}
}

func Use(vals ...interface{}) {
    for _, val := range vals {
        _ = val
    }
}

func yobaSync(s *Server, ctx context.Context, receiveFromClient chan byte,receiveFromServer chan byte, sendToClient chan byte,  sendToServer chan byte)  (*well.Environment, chan bool) {
	unfuck := make(chan bool, 1)

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		for {
	    	select {
	    		case shit := <- receiveFromServer:
	    			s.Logger.Info("Sync2", well.FieldsFromContext(ctx))
	    			sendToClient <- shit
	    		case shit := <- receiveFromClient:
	    			s.Logger.Info("Sync1", well.FieldsFromContext(ctx))
	    			sendToServer <- shit
	    	}
	    }
		return nil
	})
	return env, unfuck
}


func makeReadChannel(s *Server, ctx context.Context, conn net.Conn, destConn net.Conn, bufferedReader *bufio.Reader,bufferedWriter *bufio.Writer, tx chan byte, halfass chan bool) (*well.Environment, chan bool) {
	unfuck := make(chan bool, 10)

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		halfassed := false
		var total int64
		total = 0
		for {
    		select {
        		case <- unfuck:
        			return errors.New("unfucked")
        		default:
        			s.Logger.Info("default:" +  conn.RemoteAddr().String(), well.FieldsFromContext(ctx))

        			//if _, ok := destConn.(netutil.HalfCloser); ok {
						//hc.CloseRead()
						//hc.CloseWrite()
						//destConn.Close()
						//unfuck <- true
						//return errors.New("FIN")
					//}
					//if _, ok := conn.(netutil.HalfCloser); ok {
					//	return errors.New("FIN")
					//}
        			//data, err := bufferedReader.ReadByte();
        			buf := s.pool.Get().([]byte)
					total2, err := io.CopyBuffer(conn, destConn,buf)
					total = total + total2
					s.pool.Put(buf)
					if (total == 0) {
						s.Logger.Info("makeReadChannelREDO" + strconv.FormatInt(total,10),well.FieldsFromContext(ctx))
						if hc, ok := destConn.(netutil.HalfCloser); ok {
							hc.CloseRead()
						}
						return errors.New("RETRY")
					} else {
						if (!halfassed) {
							halfass <- true
							close(halfass)
						}
					}
        			//s.Logger.Info("ENDdefault:" + strconv.FormatInt(total,10) + " " + conn.RemoteAddr().String(), well.FieldsFromContext(ctx))
					if err != nil {
						s.Logger.Error("Error from ReadByte():" + err.Error() + " " + conn.RemoteAddr().String(), well.FieldsFromContext(ctx))
						return err

					} else {
						s.Logger.Info("GotByte from ReadByte():" +  conn.RemoteAddr().String(), well.FieldsFromContext(ctx))
					}

					if hc, ok := destConn.(netutil.HalfCloser); ok {
						hc.CloseRead()
						hc.CloseWrite()
						destConn.Close()
						unfuck <- true
						return errors.New("FIN")
					}
					
					//if _, ok := conn.(netutil.HalfCloser); ok {
					//	return errors.New("FIN")
					//}
					
				
					//tx <- data
				}
			}
		//conn.Close()
		return nil
	})
	return env, unfuck
}

func makeReadTeeChannel(s *Server, ctx context.Context, conn net.Conn, destConn net.Conn, teeReader io.Reader, origReader *bufio.Reader, tx chan byte) (*well.Environment,chan bool, chan bool) {
	unfuck := make(chan bool, 100)
	halfass := make(chan bool, 100)

	

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		halfassed := false
		i := 0
		for {
    		select {
        		case <- unfuck:
        			return errors.New("unfucked")
        		case <- halfass:
        			if (!halfassed) {
	        			s.Logger.Info("Switched to half-ass packet mode", well.FieldsFromContext(ctx))
	        			teeReader = bufio.NewReader(conn)
	        			halfassed = true
        			}
        		default:
        			buf := s.pool.Get().([]byte)
					total, err := io.CopyBuffer(destConn, teeReader,buf)
					s.pool.Put(buf)

					if (!halfassed) {
						continue
					}

					//if (total == 0) {
					//	s.Logger.Info("makeReadTeeChannelREDO" + strconv.FormatInt(total,10),well.FieldsFromContext(ctx))
					if _, ok := conn.(netutil.HalfCloser); ok {
					//	hc.CloseRead()
					//	hc.CloseWrite()
					//	conn.Close()
						return errors.New("FIN")
					}
					//}
        			//buf := make([]byte, 1)
        			//_, err := io.ReadFull(teeReader, buf);
        			//data, err := teeReader.ReadByte();
        			if (err != nil) {
						s.Logger.Info("ClientToServer" + strconv.FormatInt(total,10) + ":" + err.Error(),well.FieldsFromContext(ctx))
					} else {
						s.Logger.Info("ClientToServer" + strconv.FormatInt(total,10),well.FieldsFromContext(ctx))
					}
        			i++
        			//byteToInt, _ := strconv.Atoi(string(data))
					if err != nil {
						s.Logger.Error("Error from ReadByte():" + err.Error(), well.FieldsFromContext(ctx))
						return err
					}
					s.Logger.Info("DEFAULT_END[" + "]:" + strconv.Itoa(i) + ":" +  conn.RemoteAddr().String(), well.FieldsFromContext(ctx))
					//tx <- data

				}
			}
		//conn.Close()
		return nil
	})
	return env,unfuck,halfass
}

func makeWriteChannel(s *Server, ctx context.Context, conn net.Conn, bufferedWriter *bufio.Writer, rx chan byte) (*well.Environment,chan bool) {
	unfuck := make(chan bool)
	env := well.NewEnvironment(ctx)
	
	env.Go(func(ctx context.Context) error {
		i := 0
		for {
    		select {
        		case <- unfuck:
        			return errors.New("unfucked")
        		case data := <- rx:
        			err := bufferedWriter.WriteByte(data)
        			i++
        			byteToInt, _ := strconv.Atoi(string(data))
        			s.Logger.Info("WroteByte[" + strconv.Itoa(byteToInt) + "]:" + strconv.Itoa(i) + ":" +  conn.RemoteAddr().String(), well.FieldsFromContext(ctx))
					if err != nil {
						s.Logger.Error("Error from WriteByte("+string(data)+"):" + err.Error(), well.FieldsFromContext(ctx))
						return err
					}
					
				}
			}
		//conn.Close()
		return nil
	})
	return env,unfuck
}



func handleConnectionPipe( s *Server, ctx context.Context, conn net.Conn, destConn net.Conn, i int, lastBuf bytes.Buffer) (bytes.Buffer) {

		st := time.Now()
		env := well.NewEnvironment(ctx)
		env1 := well.NewEnvironment(ctx)

		//shit := well.FieldsFromContext(ctx)
		//shit["client_addr"] = "TEST"
		var bufferRead bytes.Buffer

		bufreader := bufio.NewReader(conn)
		reader := io.TeeReader(conn, &bufferRead)

		if i > 0 {
			bufreader = bufio.NewReader(&lastBuf)
			reader = io.TeeReader(bufreader, &bufferRead)
		}

		
		env1.Go(func(ctx context.Context) error {

			buf := s.pool.Get().([]byte)

			total, err := io.CopyBuffer(destConn, reader, buf)
			//total, err = reader.Read()
			if (err != nil && err.Error() == "CANCEL") {
				s.Logger.Error("CATCHEDRETRYINNER", well.FieldsFromContext(ctx))
			}
			//s.pool.Put(buf)
			s.Logger.Info("NOTCLOSING_destConn_Writer_env1",well.FieldsFromContext(ctx))
			
			if hc, ok := destConn.(netutil.HalfCloser); ok {
				hc.CloseWrite()
			}
			s.Logger.Info("NOTCLOSING_conn_Reader_env1",well.FieldsFromContext(ctx))
			
			if hc, ok := conn.(netutil.HalfCloser); ok {
				hc.CloseRead()
			}

			if (err != nil) {
				s.Logger.Info("ClientToServer" + strconv.FormatInt(total,10) + ":" + err.Error(),well.FieldsFromContext(ctx))
			} else {
				s.Logger.Info("ClientToServer" + strconv.FormatInt(total,10),well.FieldsFromContext(ctx))
			}
			return err
	

		})
		env.Go(func(ctx context.Context) error {
			buf := s.pool.Get().([]byte)
			total, err := io.CopyBuffer(conn, destConn, buf)
			s.pool.Put(buf)
			if (total == 0) {
				//s.Seek(0,io.SeekStart)
				s.Logger.Info("REDO" + strconv.FormatInt(total,10),well.FieldsFromContext(ctx))
				/*if hc, ok := destConn.(netutil.HalfCloser); ok {
					hc.CloseRead()
				}*/
				return errors.New("RETRY")
				//destConn = s.handleSOCKS4(ctx, conn, preamble[1])
			} else {
				if hc, ok := conn.(netutil.HalfCloser); ok {
					hc.CloseWrite()
				}
				if hc, ok := destConn.(netutil.HalfCloser); ok {
					hc.CloseRead()
				}
				
				if (err != nil) {
					s.Logger.Info("ServerToClient" + strconv.FormatInt(total,10) + ":" + err.Error(),well.FieldsFromContext(ctx))
				} else {
					s.Logger.Info("ServerToClient" + strconv.FormatInt(total,10),well.FieldsFromContext(ctx))
				}
			}
			return err
		})
		env.Stop()
		err := env.Wait()
		var result bytes.Buffer

		if (err != nil && "RETRY" == err.Error()) {
			result = bufferRead
			s.Logger.Error("CATCHEDRETRY", well.FieldsFromContext(ctx))
			env1.Cancel(errors.New("CANCEL"))
			destConn.Close()
			err = env1.Wait()
			//handleConnectionSocks(s, ctx, conn, destConn net.Conn, preamble [2]byte, i int, reader io.Reader, buf bytes.Buffer)
		} else {
			env1.Stop()
			err = env1.Wait()
		}
		

		fields := well.FieldsFromContext(ctx)
		fields["elapsed"] = time.Since(st).Seconds()
		if err != nil {
			fields[log.FnError] = err.Error()
			s.Logger.Error("proxy ends with an error", fields)
			//continue
			result = bufferRead
		}
		if s.SilenceLogs {
			s.Logger.Debug("proxy ends", fields)
		} else {
			s.Logger.Info("proxy ends", fields)
		}
		return result

}










/*
func ReadConnWithSelect(conn net.Conn) (x int, err error) {
 timeout := time.NewTimer(time.Microsecond * 500)

 select {
 case x = conn.Read():
  return x, nil
 case <-timeout.C:
  return 0, errors.New("read time out")
 }
}



func ReadWithSelect(ch chan int) (x int, err error) {
 timeout := time.NewTimer(time.Microsecond * 500)

 select {
 case x = <-ch:
  return x, nil
 case <-timeout.C:
  return 0, errors.New("read time out")
 }
}

func WriteChWithSelect(ch chan int) error {
 timeout := time.NewTimer(time.Microsecond * 500)

 select {
 case ch <- 1:
  return nil
 case <-timeout.C:
  return errors.New("write time out")
 }
}*/