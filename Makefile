VERSION= 1.0.1
LDFLAGS= -ldflags "-X main.version=${VERSION}"
RPM_BUILD_PATH= ~/rpmbuild
RPM_BUILD_ROOT= ${RPM_BUILD_PATH}/BUILDROOT

default: build

run: build
	cd receiver; ./rsa-nw-syslog-receiver

depends:
	go get -d ./...

build: depends
	cd receiver; go build $(LDFLAGS) -o rsa-nw-syslog-receiver

clean:
	rm -f receiver/rsa-nw-syslog-receiver
	rm -Rf ${RPM_BUILD_PATH}

install: build
	mkdir -p ${RPM_BUILD_ROOT}
	mkdir -p ${RPM_BUILD_ROOT}/usr/local/bin/
	mkdir -p ${RPM_BUILD_ROOT}/etc/syslogreceiver/
	mkdir -p ${RPM_BUILD_ROOT}/usr/lib/systemd/system/
	cp receiver/rsa-nw-syslog-receiver ${RPM_BUILD_ROOT}/usr/local/bin
	cp conf/syslogreceiver.conf ${RPM_BUILD_ROOT}/etc/syslogreceiver
	cp conf/rsa-nw-syslog-receiver.service ${RPM_BUILD_ROOT}/usr/lib/systemd/system

rpm: install
	mkdir -p ${RPM_BUILD_PATH}/SPECS ${RPM_BUILD_PATH}/RPMS ${RPM_BUILD_PATH}/SOURCES
	cp conf/rpmbuild/SPECS/rsa-nw-syslog-receiver.spec ${RPM_BUILD_PATH}/SPECS
	sed -i 's/%VERSION%/${VERSION}/' ${RPM_BUILD_PATH}/SPECS/rsa-nw-syslog-receiver.spec
	cp LICENSE ${RPM_BUILD_PATH}/SOURCES/license
	rpmbuild -ba ${RPM_BUILD_PATH}/SPECS/rsa-nw-syslog-receiver.spec
