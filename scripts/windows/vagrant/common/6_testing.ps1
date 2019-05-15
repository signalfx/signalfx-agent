# ensure choco in path
$env:Path = [Environment]::GetEnvironmentVariable('Path',[System.EnvironmentVariableTarget]::Machine);

choco install -y --source windowsfeatures IIS-WebServerRole
choco install -y microsoft-visual-cpp-build-tools