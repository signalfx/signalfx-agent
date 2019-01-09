# ensure choco in path
$env:Path = [Environment]::GetEnvironmentVariable('Path',[System.EnvironmentVariableTarget]::Machine);

choco install -y make `
                 git `
                 firefox `
                 vscode `
                 python `
                 vcpython27