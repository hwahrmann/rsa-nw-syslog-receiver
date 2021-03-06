# Summary

When Syslog events are forwarded to RSA Netwitness by a Syslog Relay Server, and the
Relay Server cannot rewrite the Sender address, usually only available via UDP, it looks
like all events are coming from the same host, namely the Relay Server.

A possible workaround is to Prefix the Syslog message with a special Header:
[][][originalhost][eventtime][]

This will assign the correct device.ip, in case originalhost is an IP address, or device.host,
when a hostname is specified.

Some Syslog Receivers capbable of prefixing messages are rsyslog or SyslogNG.
For others, not having this feature, the Syslog Receiver has been developed.

The Syslog Receiver listens on a given port for incoming Syslog Messages.
The messages can be in either RFC3164 or RFC5424 Format.
Custom formats are supported by specifying a Regular Expression, which extracts the hostname of the origin sender and the original message.

The events are then forwarded to a RSA Netwitness Log Decoder or to the Syslog Service on a
RSA Netwitness Log Collector.

Events, which can not be forwarded to RSA Netwitness are buffered locally. Once the Service
is up again, the events are sent.

Options are specified as command line arguments or via Configuration file.

## Features
- Collection of RFC3164 and RFC5424 syslog formats
- Forwarding of events to RSA Netwitness
- Buffering of events in case of RSA Netwitness infrastructure downtime
- Multiple Worker to allow concurrent processing of events
- Simple REST interface for stats


## Documentation
- [Configuration](/docs/config.md).

## Supported platform
- Linux

## Build
Given that the Go Language compiler (version 1.11 or aboce preferred) is installed, you can build it with:
```
go get github.com/hwahrmann/rsa-nw-syslog-receiver
cd $GOPATH/src/github.com/hwahrmann/rsa-nw-syslog-receiver

make build
```
The binary is then in the subfolder named receiver.

To check the version:
```
./rsa-nw-syslog-receiver -version
```

Altough you could specify parameters, like described in [Configuration](/docs/config.md), it is best to create a config file, a sample is in the conf folder, and star it like this:
```
./rsa-nw-syslog-receiver -config myconfig.conf
```

## Installation
You can download and install a pre-built rpm package as below ([RPM](https://github.com/hwahrmann/rsa-nw-syslog-receiver/releases)).

```
yum localinstall rsa-nw-syslog-receiver-1.0.0-1.x86_64.rpm
```

Once you installed you need to configure some basic parameters, for more information check [[Configuration](/docs/config.md):
```
/etc/syslogreceiver/syslogreceiver.conf
```
You can start the service by the below:
```
service rsa-nw-syslog-receiver start
```

## License
Licensed under the Apache License, Version 2.0 (the "License")

## Contribute
Welcomes any kind of contribution, please follow the next steps:

- Fork the project on github.com.
- Create a new branch.
- Commit changes to the new branch.
- Send a pull request.
