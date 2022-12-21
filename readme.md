# 层级分析工具
## 改动
0.0.1 首次release  
0.0.2 使用hashMap来进行循环检测, 去除进度条

## 如何使用

```
./levelman_linux_amd64 --help
  -cpu string
        write cpu profile to file
  -ee int
        下线所在列 (default 1)
  -er int
        上线所在列 (default 2)
  -f string
        输入文件路径 (default "in.csv")
  -h    是否含表头 (default true)
  -mem string
        write mem profile to file
```