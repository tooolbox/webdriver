// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type EdgeSwitches map[string]interface{}

type EdgeDriver struct {
	WebDriverCore
	//The port that EdgeDriver listens on. Default: 9515
	Port int
	//The URL path prefix to use for all incoming WebDriver REST requests. Default: ""
	BaseUrl string
	//The number of threads to use for handling HTTP requests. Default: 4
	Threads int
	//The path to use for the EdgeDriver server log. Default: ./Edgedriver.log
	LogPath string
	// Log file to dump Edgedriver stdout/stderr. If "" send to terminal. Default: ""
	LogFile string
	// Start method fails if Edgedriver doesn't start in less than StartTimeout. Default 20s.
	StartTimeout time.Duration

	path    string
	cmd     *exec.Cmd
	logFile *os.File
}

//create a new service using Edgedriver.
//function returns an error if not supported switches are passed. Actual content
//of valid-named switches is not validate and is passed as it is.
//switch silent is removed (output is needed to check if Edgedriver started correctly)
func NewEdgeDriver(path string) *EdgeDriver {
	d := &EdgeDriver{}
	d.path = path
	d.Port = 17556
	d.BaseUrl = ""
	d.Threads = 4
	d.LogPath = ""
	d.StartTimeout = 20 * time.Second
	return d
}

func (d *EdgeDriver) Start() error {
	csferr := "Edgedriver start failed: "
	if d.cmd != nil {
		return errors.New(csferr + "Edgedriver already running")
	}

	if d.LogPath != "" {
		//check if log-path is writable
		file, err := os.OpenFile(d.LogPath, os.O_WRONLY|os.O_CREATE, 0664)
		if err != nil {
			return errors.New(csferr + "unable to write in log path: " + err.Error())
		}
		file.Close()
	}

	d.url = fmt.Sprintf("http://localhost:%d%s", d.Port, d.BaseUrl)
	var switches []string
	switches = append(switches, "--port="+strconv.Itoa(d.Port))
	// switches = append(switches, "--w3c=false")
	switches = append(switches, "--jwp=true")
	// switches = append(switches, "--verbose")
	// switches = append(switches, "-log-path="+d.LogPath)
	// switches = append(switches, "-http-threads="+strconv.Itoa(d.Threads))
	// if d.BaseUrl != "" {
	// 	switches = append(switches, "-url-base="+d.BaseUrl)
	// }

	d.cmd = exec.Command(d.path, switches...)
	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		return errors.New(csferr + err.Error())
	}
	stderr, err := d.cmd.StderrPipe()
	if err != nil {
		return errors.New(csferr + err.Error())
	}
	if err := d.cmd.Start(); err != nil {
		return errors.New(csferr + err.Error())
	}
	if d.LogFile != "" {
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		d.logFile, err = os.OpenFile(d.LogFile, flags, 0640)
		if err != nil {
			return err
		}
		go io.Copy(d.logFile, stdout)
		go io.Copy(d.logFile, stderr)
	} else {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}
	if err = probePort(d.Port, d.StartTimeout); err != nil {
		return err
	}
	return nil
}

func (d *EdgeDriver) Stop() error {
	if d.cmd == nil {
		return errors.New("stop failed: Edgedriver not running")
	}
	defer func() {
		d.cmd = nil
	}()
	d.cmd.Process.Signal(os.Interrupt)
	if d.logFile != nil {
		d.logFile.Close()
	}
	return nil
}

func (d *EdgeDriver) NewSession(desired, required Capabilities) (*Session, error) {
	//id, capabs, err := d.newSession(desired, required)
	//return &Session{id, capabs, d}, err
	session, err := d.newSession(desired, required)
	if err != nil {
		return nil, err
	}
	session.wd = d
	return session, nil
}

func (d *EdgeDriver) Sessions() ([]Session, error) {
	sessions, err := d.sessions()
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].wd = d
	}
	return sessions, nil
}
