# TODO: centralize these
$depVersion = "v0.5.0"
$goVersion = "1.11.4"

# ensure choco in path
$env:Path = [Environment]::GetEnvironmentVariable('Path',[System.EnvironmentVariableTarget]::Machine);

# install golang
choco install -y golang --version $goVersion

# create go bin and pkg directories
mkdir c:\users\vagrant\go\bin -ea 0
mkdir c:\users\vagrant\go\pkg -ea 0
mkdir c:\users\vagrant\go\src -ea 0

# add go bin to path
$oldpath = (Get-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH).path
$newpath = "$oldpath;C:\GO\bin;c:\users\vagrant\go\bin"
Set-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath

# set gopath variable
[Environment]::SetEnvironmentVariable("GOPATH", "c:\users\vagrant\go", "Machine")

# install go dep
$url = "https://raw.githubusercontent.com/golang/dep/master/install.sh"
$output = "C:\tmp\dep\install.sh"
mkdir c:\tmp\dep -ea 0
(New-Object System.Net.WebClient).DownloadFile($url, $output)

& 'C:\Program Files\Git\git-bash.exe' -i -l -c "GOBIN=/C/Users/vagrant/go/bin DEP_RELEASE_TAG=$depVersion echo $DEP_RELEASE_TAG; /C/tmp/dep/install.sh"
