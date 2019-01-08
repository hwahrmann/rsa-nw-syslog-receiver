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

## License
Licensed under the Apache License, Version 2.0 (the "License")

## Contribute
Welcomes any kind of contribution, please follow the next steps:

- Fork the project on github.com.
- Create a new branch.
- Commit changes to the new branch.
- Send a pull request.
