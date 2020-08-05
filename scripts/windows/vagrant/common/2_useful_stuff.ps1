# ensure choco in path
$env:Path = [Environment]::GetEnvironmentVariable('Path',[System.EnvironmentVariableTarget]::Machine);

choco install -y --ignorepackagecodes make `
                 git `
                 firefox `
                 vscode `
                 python `
                 vcpython27 `
                 vcredist2015

choco install -y checksum

choco install -y wixtoolset --version 3.11.2
