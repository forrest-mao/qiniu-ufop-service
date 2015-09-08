#html2pdf

###简介：
基于`wkhtmltopdf`实现的将html页面转换为pdf文档的程序。

###二进制包下载

|开发环境|下载|
|-------|-----|
|Mac|http://download.gna.org/wkhtmltopdf/0.12/0.12.2/wkhtmltox-0.12.2_osx-cocoa-x86-64.pkg|

|Ufop环境|下载|
|------------|------|
|Ubuntu14.04 64位|http://download.gna.org/wkhtmltopdf/0.12/0.12.2/wkhtmltox-0.12.2_linux-trusty-amd64.deb|

###方案
使用`wkhtmltopdf`二进制文件直接实现`html`到`pdf`的转换，ufop提供转换的参数设置界面和进程调用。

|API 可选参数|对应`wkhtmltopdf`指令参数|
|------|-----------|
|gray/[1 or 0] | -g, --grayscale |
|low/[1 or 0]  | -l, --lowquality |
|orient/[Landscape or Portrait] | -O, --orientation  |
|size/[A4, A3 ...] | -s --page-size  |
|title/[Urlsafe Base64 Encoded Title] | --title   |
|collate/[1 or 0] | --collate, --no-collate   |
|copies/[1 or more] | --copies   |

其中`size`的可选值为：

|A Size|B Size|
|----|-----|
|A1|B1|
|A2|B2|
|A3|B3|
|A4|B4|
|A5|B5|
|A6|B6|
|A7|B7|
|A8|B8|

**关于`collate`参数的含义：**

这个参数在`copies>1`的情况下，表现出不同。
所谓`collate`的意思就是，当你有比如3页文档需要输出两份的时候，当`collate/1`的情况下，输出顺序为`1,2,3,1,2,3`。
当`collate/0`的情况下，输出顺序为`1,1,2,2,3,3`。默认不指定`collate`的情况下，`collate`为1。
