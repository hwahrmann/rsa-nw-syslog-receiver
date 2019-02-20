###############################################################################
# Spec file for rsa-nw-syslog-receiver
################################################################################
# Configured to be built by non-root user
################################################################################
#
# Build with the following syntax:
# rpmbuild -bb rsa-nw-syslog-receiver.spec
#
Summary: Syslog Receiver to parse RFC3164 and RFC5424 compliant messages and forward them to a RSA Netwitness Log Decoder
Name: rsa-nw-syslog-receiver
Version: %VERSION%
Release: 1.0.1
License: Apache
Group: Utilities
Packager: Helmut Wahrmann
BuildRoot: ~/rpmbuild/

%description
The Syslog Receiver can be configured via Command line or in a config file.
The config file can be specified as part of the command line.
If not present the default config in /etc/syslogreceiver/syslogreceiver.conf
will be used.

Syslog Receiver listens on the specified port for incoming messages.
It parses RFC3164 and RFC5424 compliant messages, extracts the Orignial Sender
and then forwards a special formatted message to a RSA Netwitness Log Decoder.

%prep
echo "BUILDROOT = $RPM_BUILD_ROOT"
mkdir -p $RPM_BUILD_ROOT
mv $RPM_BUILD_ROOT/../usr/ $RPM_BUILD_ROOT/
mv $RPM_BUILD_ROOT/../etc/ $RPM_BUILD_ROOT/

%files
%attr(0744, root, root) /usr/local/bin/*
%attr(0644, root, root) /etc/syslogreceiver/*
%attr(0644, root, root) /usr/lib/systemd/system/rsa-nw-syslog-receiver.service

%post
################################################################################
# Set up a sybobilc link to our new service                                    #
################################################################################
cd /etc/systemd/system/multi-user.target.wants

if [ ! -e rsa-nw-syslog-receiver.service ]
then
   ln -s /usr/lib/systemd/system/rsa-nw-syslog-receiver.service
fi

%postun
# remove installed files and links
rm /etc/systemd/system/multi-user.target.wants/rsa-nw-syslog-receiver.service

%clean
rm -rf $RPM_BUILD_ROOT/usr
rm -rf $RPM_BUILD_ROOT/etc

%changelog
* Fri Jan 04 2019 Helmut Wahrmann <helmut.wahrmann@rsa.com>
  - Inital release
