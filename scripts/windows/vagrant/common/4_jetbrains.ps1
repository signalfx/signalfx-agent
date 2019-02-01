# ensure choco in path
$env:Path = [Environment]::GetEnvironmentVariable('Path',[System.EnvironmentVariableTarget]::Machine);

# install jetbrains products
choco install -y goland `
                 pycharm