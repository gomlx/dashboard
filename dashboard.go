// Package dashboard implements a web-based dashboard for: (1) Checkpoint of a GoMLX model; (2) Training loop for
// a training model.
//
// To use in your training loop:
//
//	var *flagWebUI = flag.String("webui", "", "To configure a WebUI to train, set to a port number, or :0 to automatically "+
//		"allocate one. It will print out the URL to connect to the dashboard.")
//
//	...
//	func main() {
//		...
//		if *flagWebUI != "" {
//			trainUI := dashboard.New(*flagWebUI).
//				WithCheckpoint(checkpoint).
//				AttachToTrainLoop(train.loop).
//				Start()
//			defer trainUI.Stop()
//		}
//	}
package dashboard

import (
	"fmt"
	"net"
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

// Dashboard implements a web-based dashboard for GoMLX checkpoint and training loop.
// See package documentation for description and examples.
//
// You can create a new one with New.
type Dashboard struct {
	address   string
	server    *http.Server
	verbosity int

	templatesPath string
}

// New creates a new Dashboard with the given address.
//
// Valid address formats:
//   - "localhost:0" - Listen to the localhost interface only (no external connections) with an automatically
//     allocated port. Same (usually) as "127.0.0.1:0".
//   - ":port" - Listen on all interfaces on the specified port (e.g. ":8080")
//   - ":0" - Listen on all interfaces with an automatically allocated port
//   - "host:port" - Listen on specific interface and port (e.g. "localhost:8080")
//   - "" - Converted to "localhost:0", will automatically allocate a port on the localhost only --
//     no external connections.
//
// The returned Dashboard can be further configured. Once configured, call Start to actually start serving.
func New(address string) *Dashboard {
	if address == "" {
		address = "localhost:0"
	}
	return &Dashboard{
		address:   address,
		verbosity: 1,
	}
}

// Start serving dashboard.
func (d *Dashboard) Start() *Dashboard {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello World!")
		if err != nil {
			klog.Errorf("Failed to write response: %v", err)
		}
	})

	d.server = &http.Server{
		Addr:    d.address,
		Handler: mux,
	}

	listener, err := net.Listen("tcp", d.address)
	if err != nil {
		klog.Errorf("Failed to start listener: %v", err)
		return d
	}
	d.server = &http.Server{
		Handler: mux,
	}

	go func() {
		err := d.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			klog.Errorf("Failed to start server: %v", err)
		}
	}()

	d.address = "http://" + listener.Addr().String()
	if d.verbosity > 1 {
		fmt.Printf("Dashboard started at %s\n", d.address)
	}
	klog.V(1).Infof("Dashboard started at %s", d.address)

	return d
}

// Address provided to New if the dashboard hasn't yet started.
// After it starts, the address where it is being served -- including the protocol ("http://").
func (d *Dashboard) Address() string {
	return d.address
}

// Stop shuts down the dashboard server.
func (d *Dashboard) Stop() error {
	if d.server != nil {
		err := d.server.Close()
		d.server = nil
		if err != nil {
			err = errors.Wrapf(err, "failed to stop dashboard server")
		}
		return err
	}
	return nil
}

// WithTemplates sets a path where to read the HTML (HTMX) templates for Dashboard.
// Used only during development of Dashboard, or if you really want to customize the UI.
func (d *Dashboard) WithTemplates(path string) *Dashboard {
	d.templatesPath = path
	return d
}
