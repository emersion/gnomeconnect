# GNOMEConnect

A GNOME frontend for KDEConnect.

## Installing

```bash
go get -u github.com/emersion/gnomeconnect
cd $GOPATH/src/github.com/emersion/gnomeconnect
make install-user
```

Make sure to add `$GOPATH/bin` to your `$PATH` in `~/.profile`.

## Building

```bash
go get -u github.com/emersion/gnomeconnect
cd $GOPATH/src/github.com/emersion/gnomeconnect
make
```

## SFTP plugin

The Android app exposes a SFTP plugin, but it uses `ssh-dss`, which has been
removed by default from `ssh` due to security concerns. To be able to use it,
edit `~/.ssh/config` and add:

```
HostKeyAlgorithms=+ssh-dss
```
