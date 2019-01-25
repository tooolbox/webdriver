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

type IE11Switches map[string]interface{}

type IE11Driver struct {
	WebDriverCore
	//The port that IE11Driver listens on. Default: 9515
	Port int
	//The URL path prefix to use for all incoming WebDriver REST requests. Default: ""
	BaseUrl string
	//The number of threads to use for handling HTTP requests. Default: 4
	Threads int
	//The path to use for the IE11Driver server log. Default: ./IE11driver.log
	LogPath string
	// Log file to dump IE11driver stdout/stderr. If "" send to terminal. Default: ""
	LogFile string
	// Start method fails if IE11driver doesn't start in less than StartTimeout. Default 20s.
	StartTimeout time.Duration

	path    string
	cmd     *exec.Cmd
	logFile *os.File
}

//create a new service using IE11driver.
//function returns an error if not supported switches are passed. Actual content
//of valid-named switches is not validate and is passed as it is.
//switch silent is removed (output is needed to check if IE11driver started correctly)
func NewIE11Driver(path string) *IE11Driver {
	d := &IE11Driver{}
	d.path = path
	d.Port = 5555
	d.BaseUrl = ""
	d.Threads = 4
	d.LogPath = ""
	d.StartTimeout = 20 * time.Second
	return d
}

func (d *IE11Driver) Start() error {
	csferr := "IE11driver start failed: "
	if d.cmd != nil {
		return errors.New(csferr + "IE11driver already running")
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

func (d *IE11Driver) Stop() error {
	if d.cmd == nil {
		return errors.New("stop failed: IE11driver not running")
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

func (d *IE11Driver) NewSession(desired, required Capabilities) (*Session, error) {
	//id, capabs, err := d.newSession(desired, required)
	//return &Session{id, capabs, d}, err
	session, err := d.newSession(desired, required)
	if err != nil {
		return nil, err
	}
	session.wd = d
	return session, nil
}

func (d *IE11Driver) Sessions() ([]Session, error) {
	sessions, err := d.sessions()
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].wd = d
	}
	return sessions, nil
}
