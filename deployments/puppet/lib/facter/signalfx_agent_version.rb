# This custom fact checks for the installed agent version on windows.
# Returns empty string if the key does not exist.

Facter.add(:signalfx_agent_version) do
  confine :osfamily => :windows
  setcode do
    begin
      value = ''
      Win32::Registry::HKEY_LOCAL_MACHINE.open('SYSTEM\CurrentControlSet\Services\signalfx-agent') do |regkey|
        value = regkey['CurrentVersion']
      end
      value
    rescue
      ''
    end
  end
end
