package dockeractd

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	dockerapi "github.com/fsouza/go-dockerclient"
)

type Dockeractd struct {
	client        *dockerapi.Client
	endpoint      string
	retryInterval time.Duration
	tlsCaFile     string
	tlsCertFile   string
	tlsKeyFile    string
	tlsVerify     bool
	cmd           string
}

type optionInterface interface {
	Cmd() string
	Endpoint() string
	RetryInterval() time.Duration
	TLSVerify() bool
	TLSCertFile() string
	TLSKeyFile() string
	TLSCaFile() string
}
type Options struct {
	OptCmd           string
	OptEndpoint      string
	OptRetryInterval time.Duration
	OptTLSCaFile     string
	OptTLSCertFile   string
	OptTLSKeyFile    string
	OptTLSVerify     bool
}

func (o Options) Cmd() string                  { return o.OptCmd }
func (o Options) Endpoint() string             { return o.OptEndpoint }
func (o Options) RetryInterval() time.Duration { return o.OptRetryInterval }
func (o Options) TLSCaFile() string            { return o.OptTLSCaFile }
func (o Options) TLSCertFile() string          { return o.OptTLSCertFile }
func (o Options) TLSKeyFile() string           { return o.OptTLSKeyFile }
func (o Options) TLSVerify() bool              { return o.OptTLSVerify }

func MakeDefaultOptions() *Options {
	o := &Options{
		OptEndpoint:      "unix:///var/run/docker.sock",
		OptRetryInterval: 5 * time.Second,
	}

	if v := os.Getenv("DOCKER_HOST"); v != "" {
		o.OptEndpoint = v
	}

	if v := os.Getenv("DOCKER_CERT_PATH"); v != "" {
		ok := 0
		fn := filepath.Join(v, "cert.pem")
		if _, err := os.Stat(fn); err == nil {
			o.OptTLSCertFile = fn
			ok++
		}

		fn = filepath.Join(v, "key.pem")
		if _, err := os.Stat(fn); err == nil {
			o.OptTLSKeyFile = fn
			ok++
		}

		fn = filepath.Join(v, "ca.pem")
		if _, err := os.Stat(fn); err == nil {
			o.OptTLSCaFile = fn
			ok++
		}

		if ok == 3 {
			o.OptTLSVerify = true
		}
	}

	return o
}

func New(o optionInterface) *Dockeractd {
	if o == nil {
		o = MakeDefaultOptions()
	}

	return &Dockeractd{
		cmd:           o.Cmd(),
		endpoint:      o.Endpoint(),
		retryInterval: o.RetryInterval(),
		tlsCaFile:     o.TLSCaFile(),
		tlsCertFile:   o.TLSCertFile(),
		tlsKeyFile:    o.TLSKeyFile(),
		tlsVerify:     o.TLSVerify(),
	}
}

func (d *Dockeractd) Run() error {
	loop := true
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for loop {
		select {
		case <-sigCh:
			loop = false
			continue
		default:
		}

		err := d.attachToDocker()
		if err != nil {
			if t := d.retryInterval; t > 0 {
				time.Sleep(t)
			}

			continue
		}

		// wait for events
		events := make(chan *dockerapi.APIEvents)
		if err = d.client.AddEventListener(events); err != nil {
			log.Printf("Error receiving events: %s", err)
			continue
		}

	INNER:
		for {
			select {
			case <-sigCh:
				loop = false
				break INNER
			case ev := <-events:
				if ev == nil {
					loop = false
					break INNER
				}
				if err := d.process(ev); err != nil {
					log.Printf("Error executing script: %s", err)
				}
			}
		}
	}
	return nil
}

func (d *Dockeractd) attachToDocker() error {
	var client *dockerapi.Client
	var err error

	log.Printf("Attaching to %s", d.endpoint)
	if d.tlsVerify {
		log.Printf("Enabling TLS...")
		client, err = dockerapi.NewTLSClient(
			d.endpoint,
			d.tlsCertFile,
			d.tlsKeyFile,
			d.tlsCaFile,
		)
	} else {
		client, err = dockerapi.NewClient(d.endpoint)
	}

	if err != nil {
		log.Printf("err =%s\n", err)
		return err
	}

	d.client = client
	return nil
}

func (d *Dockeractd) process(ev *dockerapi.APIEvents) error {
	// It's okay if we can't get the container
	container, _ := d.client.InspectContainer(ev.ID)

	payload := struct {
		Event     *dockerapi.APIEvents
		Container *dockerapi.Container
	}{
		ev,
		container,
	}

	cmd := exec.Command(d.cmd)
	buf := &bytes.Buffer{}
	cmd.Stdin = buf
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	enc := json.NewEncoder(buf)
	enc.Encode(payload)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}