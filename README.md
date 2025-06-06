# VSCode

在VSCode中配置`code`命令,使用快捷键`command + shift + p`输入`shell code`,将`code`命令添加到Path中


# Build 

```bash
go build .
# GOOS=darwin GOARCH=amd64 go build -o tm .
ln -si $(pwd)/tm /opt/homebrew/bin/tm
mkdir $HOME/.tm && echo "path=/path/of/notes" > $HOME/.tm/config # 初始化文档目录
```




