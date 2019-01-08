# Syslog Receiver configuration

## Format

A config file is a plain text file in [YAML](https://en.wikipedia.org/wiki/YAML) format. Configuration arguments can also be specified
 via the command line. in case a key is in the config file and command line, the argument from the command line is taken.

### config file
```
key: value
```
### command line
```
-key value
```
## Configuration Keys
The Syslog Receiver configuration supports the following keys. If a key is not specified the default is taken.

|Key                     | Default                        | Description                                      |
|------------------------| -------------------------------|--------------------------------------------------|
|verbose                 | false                          | log output to stdout                             |
|pid-file                | /var/run/vflow.pid             | file in which server should write its process ID |
|logdecoder              | 127.0.0.1                      | The address of the RSA Netwitness Log Decoder    |
|logdecoderprotocol      | tcp                            | The protocol to send the syslog. tcp or udp      |
|listenport              | 5514                           | The port to listen for incoming syslog events    |
|listenprotocol          | tcp                            | The port to listen for incoing syslog events     |
|workers                 | 1                              | The number of workers to process incoming events |
|stats-enabled           | true                           | enable the REST stats server                     |
|stats-hhtp-port         | 8081                           | the REST stats server port                       |
|rfc3164                 | see below                      | The regex to parse RFC3164 syslog events         |
|rfc5424                 | see below                      | The regex to parse RFC5424 syslog events         |


The default configuration path is /etc/syslogreceiver/syslogreceiver.conf but you can change it as below:
```
rsa-nw-syslog-receiver -config /usr/local/etc/syslogreceiver.conf
```
To show version information use:
```
rsa-nw-syslog-receiver -version
```

## Some words about Regex parsing

The pattern to parse RFC3164 events is:
```
^(?P<time>[A-Z][a-z][a-z]\\s{1,2}\\d{1,2}\\s\\d{2}[:]\\d{2}[:]\\d{2})\\s(?P<host>[\\w][\\w\\d\\.@-]*)\\s(?P<message>.*)$
```
while a RFC5424 event is parsed as:
```
^[1-9]\\d{0,2} (?P<time>(\\d{4}[-]\\d{2}[-]\\d{2}[T]\\d{2}[:]\\d{2}[:]\\d{2}(?:\\.\\d{1,6})?(?:[+-]\\d{2}[:]\\d{2}|Z)?)|-)\\s(?P<host>([\\w][\\w\\d\\.@-]*)|-)\\s(?P<ident>[^ ]+)\\s(?P<pid>[-0-9]+)\\s(?P<msgid>[^ ]+)\\s?(?P<extradata>(\\[(.*)\\]|[^ ]))?\\s(?P<message>.*)$
```
Please note that the "\" needs to be escaped using "\\".

If a custom Regex pattern is used in the config file, it is important to have 2 named groups:
"<host>" specifies the original Sender
"<message>" specifies the original messages

If the specified Regex pattern does not match, the message is forwarded as-is, with the Syslog Relay Server as the originating host. 
