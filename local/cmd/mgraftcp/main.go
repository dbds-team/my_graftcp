package main

// #cgo LDFLAGS: -L../../.. -lgraftcp
//
// #include <stdlib.h>
//
// static char **alloc_string_slice(int len) {
//              return calloc(len, sizeof(char*));
// }
//
// int client_prepare(int argc, char **argv);
// int client_trace(void);
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/hmgle/graftcp/local"
	"github.com/pborman/getopt/v2"
)

func progName() string {
	name := filepath.Base(os.Args[0])
	if name == "" {
		return "graftcp"
	}
	return name
}

func appErrorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, progName()+": "+format, a...)
}

func clientPrepare(args []string) int {
	argc := C.int(len(args))

	argv := (**C.char)(C.alloc_string_slice(argc + 1))
	if argv == nil {
		appErrorf("failed to allocate argv\n")
		return 1
	}
	defer C.free(unsafe.Pointer(argv))

	argvSlice := unsafe.Slice(argv, len(args)+1)
	for i, arg := range args {
		argvSlice[i] = C.CString(arg)
		defer C.free(unsafe.Pointer(argvSlice[i]))
	}

	returnValue := C.client_prepare(argc, argv)
	return int(returnValue)
}

func clientTrace() int {
	return int(C.client_trace())
}

var version = "v0.9.0"

var cfg = defaultConfig()

func init() {
	cfg.registerFlags()
}

func main() {
	getopt.Parse()
	if cfg.showVersion {
		fmt.Printf("%s version %s\n", progName(), version)
		return
	}
	args := getopt.Args()
	if len(args) == 0 || cfg.help {
		getopt.Usage()
		fmt.Fprintf(os.Stderr, `
Examples:
  %[1]s --socks5 127.0.0.1:1080 curl https://baidu.com
  %[1]s --enable-dns --dns-server 1.1.1.1:53 curl https://baidu.com
  %[1]s -d --debug-log-file graftcp-debug.log --enable-dns curl https://baidu.com

Linux only (ptrace required). DNS proxying is opt-in and uses DNS-over-TCP
through the selected SOCKS5/HTTP proxy, or direct in direct mode.
`, progName())
		return
	}

	retCode := 0
	defer func() { os.Exit(retCode) }()

	if err := cfg.parseConfigFile(cfg.configFile); err != nil {
		appErrorf("%v\n", err)
		retCode = 1
		return
	}
	if err := cfg.resolve(); err != nil {
		appErrorf("%v\n", err)
		retCode = 1
		return
	}
	if cfg.enableDebugLog {
		logFile, err := openDebugLog(cfg.debugLogFile)
		if err != nil {
			appErrorf("open debug log: %v\n", err)
			retCode = 1
			return
		}
		defer logFile.Close()
		appLogger.Infof("proxy mode=%s DNS=%t upstream=%s UDP=%t", cfg.selectProxyMode, cfg.dnsProxy, cfg.dnsServer, cfg.udpProxy)
	}

	l, err := local.NewLocalListener(":0")
	if err != nil {
		appErrorf("%v\n", err)
		retCode = 1
		return
	}
	if err := l.SetSelectMode(cfg.selectProxyMode); err != nil {
		appErrorf("%v\n", err)
		retCode = 1
		return
	}
	if err := l.ConfigureProxy(cfg.socks5Addr, cfg.socks5User, cfg.socks5Pwd, cfg.httpProxyAddr); err != nil {
		appErrorf("%v\n", err)
		retCode = 1
		return
	}
	activeRegistry = l.Registry()
	activeUDPRegistry = l.UDPRegistry()

	ln, err := l.StartListen()
	if err != nil {
		appErrorf("l.StartListen err: %s\n", err.Error())
		retCode = 1
		return
	}
	defer ln.Close()

	dnsPort := 0
	var dnsProxy *local.DNSProxy
	if cfg.dnsProxy {
		var port int
		dnsProxy, port, err = l.ListenDNSProxy(cfg.dnsServer)
		if err != nil {
			appErrorf("%v\n", err)
			retCode = 1
			return
		}
		defer dnsProxy.Close()
		dnsPort = port
	}
	udpPort := 0
	var udpProxy *local.UDPProxy
	if cfg.udpProxy {
		var port int
		udpProxy, port, err = l.ListenUDPProxy()
		if err != nil {
			appErrorf("%v\n", err)
			retCode = 1
			return
		}
		defer udpProxy.Close()
		udpPort = port
	}

	_, faddr := l.GetFAddr()

	// Linux ties ptrace ownership to the thread that forks the tracee.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	retCode = clientPrepare(cfg.clientArgs(faddr.Port, dnsPort, udpPort, args))
	if retCode != 0 {
		return
	}

	go l.StartService(ln)
	if dnsProxy != nil {
		dnsProxy.Start()
	}
	if udpProxy != nil {
		udpProxy.Start()
	}

	retCode = clientTrace()
}
