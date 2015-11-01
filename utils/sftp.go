package utils

import (
	"io"
	"os/exec"
	"strconv"
)

func MountSftp(host string, port int, username string, password string) error {
	location := "sftp://" + username + "@" + host + ":" + strconv.Itoa(port)

	cmd := exec.Command("gvfs-mount", location)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	io.WriteString(stdin, password+"\n")
	stdin.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	cmd = exec.Command("nautilus", location+"/storage")
	cmd.Start()

	return nil
}
