%global __os_install_post %{nil}

Name: signalfx-agent
Version: %{_version}
Release: %{_release}
Summary: The SignalFx metric collection agent 

License: Apache 2.0
URL: https://github.com/signalfx/signalfx-agent

# We bundle all dependencies so don't have rpmbuild try and figure them out.
AutoReqProv: no
Requires: shadow-utils, libcap
BuildRequires: systemd
Provides: signalfx-agent

%description


%prep


%build


%install

install -d -m 755 $RPM_BUILD_ROOT/usr/lib/signalfx-agent
cp -a %{_sourcedir}/signalfx-agent/* $RPM_BUILD_ROOT/usr/lib/signalfx-agent/

install -d -m 755 $RPM_BUILD_ROOT/%{_bindir}
ln -sf /usr/lib/signalfx-agent/bin/signalfx-agent $RPM_BUILD_ROOT/%{_bindir}/signalfx-agent

install -d $RPM_BUILD_ROOT/etc/init
install -p -m 644 %{_sourcedir}/signalfx-agent.upstart $RPM_BUILD_ROOT/etc/init/signalfx-agent.conf

install -d $RPM_BUILD_ROOT/%{_unitdir}
install -p -m 644 %{_sourcedir}/systemd/signalfx-agent.service $RPM_BUILD_ROOT/%{_unitdir}/signalfx-agent.service

install -d $RPM_BUILD_ROOT/%{_tmpfilesdir}
install -p -m 644 %{_sourcedir}/systemd/signalfx-agent.tmpfile $RPM_BUILD_ROOT/%{_tmpfilesdir}/signalfx-agent.conf

install -d -m 755 $RPM_BUILD_ROOT/etc/signalfx
install -p -m 644 %{_sourcedir}/agent.yaml $RPM_BUILD_ROOT/etc/signalfx/agent.yaml

install -d $RPM_BUILD_ROOT/%{_mandir}/man1/
install -p -m 644 %{_sourcedir}/signalfx-agent.1 $RPM_BUILD_ROOT/%{_mandir}/man1/signalfx-agent.1

%files

%config /etc/signalfx/agent.yaml
/usr/lib/signalfx-agent
/etc/init/signalfx-agent.conf
/%{_unitdir}/signalfx-agent.service
/%{_bindir}/signalfx-agent
/%{_tmpfilesdir}/signalfx-agent.conf
/%{_mandir}/man1/signalfx-agent.1

%pre
getent passwd signalfx-agent >/dev/null || \
  useradd --system --home-dir /usr/lib/signalfx-agent --no-create-home --shell /sbin/nologin signalfx-agent

%post
%systemd_post signalfx-agent.service
%tmpfiles_create %{_tmpfilesdir}/signalfx-agent.conf
setcap CAP_SYS_PTRACE,CAP_DAC_READ_SEARCH=+eip /usr/lib/signalfx-agent/bin/signalfx-agent

%preun
%systemd_preun signalfx-agent.service

%postun
%systemd_postun_with_restart signalfx-agent.service


%changelog
