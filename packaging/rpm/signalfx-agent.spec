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

install -d $RPM_BUILD_ROOT/%{_unitdir}
install -p -m 644 %{_sourcedir}/systemd/signalfx-agent.service $RPM_BUILD_ROOT/%{_unitdir}/signalfx-agent.service

install -d $RPM_BUILD_ROOT/%{_tmpfilesdir}
install -p -m 644 %{_sourcedir}/systemd/signalfx-agent.tmpfile $RPM_BUILD_ROOT/%{_tmpfilesdir}/signalfx-agent.conf
install -p -m 755 %{_sourcedir}/signalfx-agent.init $RPM_BUILD_ROOT/%{_tmpfilesdir}/signalfx-agent.init

install -d -m 755 $RPM_BUILD_ROOT/etc/signalfx
install -p -m 644 %{_sourcedir}/agent.yaml $RPM_BUILD_ROOT/etc/signalfx/agent.yaml

install -d $RPM_BUILD_ROOT/%{_mandir}/man1/
install -p -m 644 %{_sourcedir}/signalfx-agent.1 $RPM_BUILD_ROOT/%{_mandir}/man1/signalfx-agent.1

%files

%config(noreplace) /etc/signalfx/agent.yaml
/usr/lib/signalfx-agent
/%{_unitdir}/signalfx-agent.service
/%{_bindir}/signalfx-agent
/%{_tmpfilesdir}/signalfx-agent.conf
/%{_tmpfilesdir}/signalfx-agent.init
/%{_mandir}/man1/signalfx-agent.1

%pre
getent passwd signalfx-agent >/dev/null || \
  useradd --system --home-dir /usr/lib/signalfx-agent --no-create-home --shell /sbin/nologin signalfx-agent

%post
if [ $1 -ge 1 ] ; then
  if command -v systemctl; then
    # Force it enabled since this is critical for monitoring
    systemctl --no-reload enable signalfx-agent.service >/dev/null 2>&1 || :
  fi
fi
%tmpfiles_create %{_tmpfilesdir}/signalfx-agent.conf
setcap CAP_SYS_PTRACE,CAP_DAC_READ_SEARCH=+eip /usr/lib/signalfx-agent/bin/signalfx-agent

%preun
if command -v systemctl; then
  %systemd_preun signalfx-agent.service
else
  if [ $1 -eq 0 ]; then
    /sbin/chkconfig --del signalfx-agent
    if [ -e /%{_initddir}/signalfx-agent ]; then
      rm -f /%{_initddir}/signalfx-agent
    fi
  fi
fi

%postun
if command -v systemctl; then
  %systemd_postun_with_restart signalfx-agent.service
fi

%posttrans
if ! command -v systemctl; then
  %tmpfiles_create %{_tmpfilesdir}/signalfx-agent.init
  if [ -e /%{_initddir}/signalfx-agent ]; then
    rm -f /%{_initddir}/signalfx-agent
  fi
  cp -a %{_tmpfilesdir}/signalfx-agent.init /%{_initddir}/signalfx-agent
  /sbin/service signalfx-agent restart > /dev/null 2>&1
  /sbin/chkconfig --add signalfx-agent
fi

%changelog
