Set-ExecutionPolicy Bypass -Scope Process -Force; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))

# temporarily enable windows update agent to upgrade powershell
Set-Service wuauserv -StartupType manual
Start-Service wuauserv

# let the service start
Start-Sleep -Seconds 5

choco install -y dotnet4.5.2 `
                 powershell

Set-Service wuauserv -StartupType disabled
Stop-Service wuauserv
