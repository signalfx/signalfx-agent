# ensure code in path
$env:Path = [Environment]::GetEnvironmentVariable('Path',[System.EnvironmentVariableTarget]::Machine);

# install extensions
code --install-extension ms-vscode.go
code --install-extension ms-python.python
code --install-extension peterjausovec.vscode-docker
code --install-extension davidanson.vscode-markdownlint
code --install-extension ms-vscode.powershell
code --install-extension minhthai.vscode-todo-parser