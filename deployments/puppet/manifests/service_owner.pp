# Sets the user/group for the signalfx-agent service.
# If the user or group does not exist, they will be created.
class signalfx_agent::service_owner ($service_name, $service_user, $service_group) {

  if $service_group == 'signalfx-agent' or $service_group in split($::local_groups, ',') {
    group { $service_group:
      noop => true,
    }
  }
  else {
    group { $service_group:
      ensure => present,
      system => true,
    }
  }

  if $service_user == 'signalfx-agent' or $service_user in split($::local_users, ',') {
    user { $service_user:
      noop => true,
    }
  }
  else {
    $shell = $::osfamily ? {
      'debian' => '/usr/sbin/nologin',
      default  => '/sbin/nologin',
    }
    user { $service_user:
      ensure => present,
      system => true,
      shell  => $shell,
      groups => $service_group,
    }
  }

  case $::service_provider {
    'systemd': {
      $tmpfile_path = "/etc/tmpfiles.d/${service_name}.conf"
      $tmpfile_dir = $tmpfile_path.split('/')[0, - 2].join('/')

      $override_path = "/etc/systemd/system/${service_name}.service.d/service-owner.conf"
      $override_dir = $override_path.split('/')[0, - 2].join('/')

      Package[$service_name] ~> Group[$service_group] ~> User[$service_user]

      ~> exec { 'systemctl stop signalfx-agent':
        path        => '/bin:/sbin:/usr/bin:/usr/sbin',
        refreshonly => true,
      }

      ~> file { [$tmpfile_dir, $override_dir]:
        ensure => directory,
      }

      ~> file {
        $tmpfile_path:
          ensure  => file,
          content => "D /run/${service_name} 0755 ${service_user} ${service_group} - -",
        ;
        $override_path:
          ensure => file,
        ;
      }

      ~> file_line {
        $override_path:
          path  => $override_path,
          line  => '[Service]',
          match => '^[Service]',
        ;
        'set-service-user':
          path    => $override_path,
          line    => "User=${service_user}",
          match   => '^User=',
          after   => '^[Service]',
          require => File_Line[$override_path],
        ;
        'set-service-group':
          path    => $override_path,
          line    => "Group=${service_group}",
          match   => '^Group=',
          after   => '^User=',
          require => File_Line['set-service-user'],
        ;
      }

      ~> exec { ["systemd-tmpfiles --create --remove ${tmpfile_path}", 'systemctl daemon-reload']:
        path        => '/bin:/sbin:/usr/bin:/usr/sbin',
        returns     => [0],
        refreshonly => true,
      }

      ~> Service[$service_name]
    }
    default: {
      $initd_path = "/etc/init.d/${service_name}"
      $default_path = "/etc/default/${service_name}"
      $run_path = "/var/run/${service_name}"
      $log_path = "/var/log/${service_name}.log"

      Package[$service_name] ~> Group[$service_group] ~> User[$service_user]

      ~> exec { 'service signalfx-agent stop':
        path        => '/bin:/sbin:/usr/bin:/usr/sbin',
        refreshonly => true,
      }

      ~> file {
        $initd_path:
          ensure => file,
        ;
        $default_path:
          ensure => file,
        ;
        $run_path:
          ensure  => directory,
          owner   => $service_user,
          recurse => true,
        ;
        $log_path:
          ensure => file,
          owner  => $service_user,
        ;
      }

      ~> file_line {
        # Update old service scripts to source vars from $default_path.
        'patch-initd-service-1':
          path  => $initd_path,
          line  => "[ -r ${default_path} ] && . ${default_path}",
          after => '^logfile=',
        ;
        'set-service-user':
          path  => $default_path,
          line  => "user=${service_user}",
          match => '^user=',
        ;
        'set-service-group':
          path  => $default_path,
          line  => "group=${service_group}",
          match => '^group=',
        ;
      }

      # Update old service scripts that explicitly set file ownership.
      ~> exec { 'patch-initd-service-2':
        path    => '/bin:/sbin:/usr/bin:/usr/sbin',
        command => "sed -i 's/chown\\(.*\\)signalfx-agent/chown\\1\$user/' ${initd_path}",
        onlyif  => "test -f ${initd_path} && grep 'chown.*signalfx-agent' ${initd_path}",
      }

      ~> Service[$service_name]
    }
  }
}
