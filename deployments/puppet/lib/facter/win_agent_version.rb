# This custom fact checks for the installed agent version on windows.

Facter.add(:win_agent_version) do
  confine :osfamily => :windows
  setcode do
    version = ''
    if File.exists?("C:\\ProgramData\\SignalFxAgent\\version.txt")
      version = File.read("C:\\ProgramData\\SignalFxAgent\\version.txt")
    end
  end
end
