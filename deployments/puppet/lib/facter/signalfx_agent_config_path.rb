# This custom fact checks for the agent config path on windows.
# Returns empty string if the key or the path does not exist.

Facter.add(:signalfx_agent_config_path) do
  confine :osfamily => :windows
  setcode do
    begin
      value = ''
      Win32::Registry::HKEY_LOCAL_MACHINE.open('SYSTEM\CurrentControlSet\Services\signalfx-agent') do |regkey|
        value = regkey['ConfigPath']
      end
      if value and !File.exists?(value)
        value = ''
      end
      value
    rescue
      ''
    end
  end
end
