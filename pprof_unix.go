//go:build !windows

package main

import (
	"net"
	"net/http"
	"net/http/pprof"
	"os"
)

// servePprof starts an HTTP server running the pprof goroutine handler on a local unix domain socket. As described in
// https://github.com/coder/coder/issues/14726 it appears this process is sometimes hanging, unable to exit cleanly,
// and this prevents additional Coder builds that try to reinstall this provider.  A goroutine dump should allow us to
// determine what is hanging.
//
// This function is best-effort, and just returns early if we fail to set up the directory/listener. We don't want to
// block the normal functioning of the provider.
func servePprof() {
	// Coder runs terraform in a per-build subdirectory of the work directory.  The per-build subdirectory uses a
	// generated name and is deleted at the end of a build, so we want to place our unix socket up one directory level
	// in the provisionerd work directory, so we can connect to it from provisionerd.
	err := os.Mkdir("../.coder", 0o700)
	if err != nil && !os.IsExist(err) {
		return
	}

	// remove the old file, if it exists. It's probably from the last run of the provider
	if _, err = os.Stat("../.coder/pprof"); err == nil {
		if err = os.Remove("../.coder/pprof"); err != nil {
			return
		}
	}
	l, err := net.Listen("unix", "../.coder/pprof")
	if err != nil {
		return
	}
	mux := http.NewServeMux()
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	srv := http.Server{Handler: mux}
	go srv.Serve(l)
	// We just leave the server and domain socket up forever. Go programs exit when the `main()` function returns, so
	// this won't block exiting, and it ensures the pprof server stays up for the entire lifetime of the provider.
}
