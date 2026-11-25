package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// Exercise the actual CLI -> ptrace -> UDP/53 -> DNS-over-TCP -> SOCKS5
// pipeline with curl's system resolver. No public DNS or Internet is needed.
func TestCurlDNSProxyEndToEnd(t *testing.T) {
	curl, err := exec.LookPath("curl")
	if err != nil {
		t.Skip("curl is required for the DNS end-to-end test")
	}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "graftcp-dns-ok")
	}))
	defer httpServer.Close()
	httpAddr := strings.TrimPrefix(httpServer.URL, "http://")
	_, httpPort, _ := net.SplitHostPort(httpAddr)

	queries := make(chan string, 16)
	dns := startTestTCPService(t, func(conn net.Conn) {
		var size [2]byte
		if _, err := io.ReadFull(conn, size[:]); err != nil {
			return
		}
		data := make([]byte, binary.BigEndian.Uint16(size[:]))
		if _, err := io.ReadFull(conn, data); err != nil {
			return
		}
		var query dnsmessage.Message
		if err := query.Unpack(data); err != nil || len(query.Questions) != 1 {
			return
		}
		q := query.Questions[0]
		queries <- q.Name.String()
		reply := dnsmessage.Message{
			Header:    dnsmessage.Header{ID: query.ID, Response: true, RecursionAvailable: true},
			Questions: query.Questions,
		}
		if q.Type == dnsmessage.TypeA {
			reply.Answers = []dnsmessage.Resource{{
				Header: dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 60},
				Body:   &dnsmessage.AResource{A: [4]byte{127, 0, 0, 1}},
			}}
		}
		wire, err := reply.Pack()
		if err != nil {
			return
		}
		binary.BigEndian.PutUint16(size[:], uint16(len(wire)))
		_, _ = conn.Write(append(size[:], wire...))
	})

	proxyTargets := make(chan string, 16)
	socks := startTestTCPService(t, func(conn net.Conn) {
		var header [4]byte
		if _, err := io.ReadFull(conn, header[:2]); err != nil || header[0] != 5 {
			return
		}
		if _, err := io.CopyN(io.Discard, conn, int64(header[1])); err != nil {
			return
		}
		_, _ = conn.Write([]byte{5, 0})
		if _, err := io.ReadFull(conn, header[:]); err != nil || header[1] != 1 {
			return
		}
		var host string
		switch header[3] {
		case 1, 4:
			size := 4
			if header[3] == 4 {
				size = 16
			}
			ip := make([]byte, size)
			if _, err := io.ReadFull(conn, ip); err != nil {
				return
			}
			host = net.IP(ip).String()
		case 3:
			if _, err := io.ReadFull(conn, header[:1]); err != nil {
				return
			}
			name := make([]byte, header[0])
			if _, err := io.ReadFull(conn, name); err != nil {
				return
			}
			host = string(name)
		default:
			return
		}
		if _, err := io.ReadFull(conn, header[:2]); err != nil {
			return
		}
		target := net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(header[:2]))))
		// A regression must never make this fixture contact a public server.
		if target != dns && target != httpAddr {
			return
		}
		proxyTargets <- target
		upstream, err := net.DialTimeout("tcp", target, 5*time.Second)
		if err != nil {
			return
		}
		defer upstream.Close()
		_ = upstream.SetDeadline(time.Now().Add(5 * time.Second))
		_, _ = conn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
		done := make(chan struct{})
		go func() {
			_, _ = io.Copy(upstream, conn)
			_ = upstream.(*net.TCPConn).CloseWrite()
			close(done)
		}()
		_, _ = io.Copy(conn, upstream)
		_ = conn.(*net.TCPConn).CloseWrite()
		<-done
	})

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], startupHelperTest, "--",
		"--select_proxy_mode", "only_socks5", "--socks5", socks,
		"--not-ignore-local", "--enable-dns", "--dns-server", dns, "--",
		curl, "-4", "--noproxy", "*", "--max-time", "8", "--fail", "--silent", "--show-error",
		"http://graftcp-dns-test.invalid:"+httpPort+"/")
	cmd.Env = append(os.Environ(), startupHelperEnv, "RES_OPTIONS=attempts:1 timeout:2")
	output, err := cmd.CombinedOutput()
	if err != nil || string(output) != "graftcp-dns-ok" {
		t.Fatalf("curl through DNS proxy: %v, output %q", err, output)
	}
	select {
	case name := <-queries:
		if name != "graftcp-dns-test.invalid." {
			t.Fatalf("DNS query = %q", name)
		}
	default:
		t.Fatal("curl succeeded without querying the configured TCP DNS upstream")
	}
	seen := map[string]bool{}
	for len(proxyTargets) > 0 {
		seen[<-proxyTargets] = true
	}
	if !seen[dns] || !seen[httpAddr] {
		t.Fatalf("SOCKS5 targets = %v, want DNS %s and HTTP %s", seen, dns, httpAddr)
	}
}

func startTestTCPService(t *testing.T, serve func(net.Conn)) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
				serve(conn)
			}()
		}
	}()
	return listener.Addr().String()
}
