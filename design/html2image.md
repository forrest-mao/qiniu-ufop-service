#html2image

###简介：
基于`wkhtmltoimage`实现的将html页面转换为图片的程序，目标图片支持`png`和`jpeg`两种格式。


###二进制包下载

|开发环境|下载|
|-------|
|Mac|http://download.gna.org/wkhtmltopdf/0.12/0.12.2/wkhtmltox-0.12.2_osx-cocoa-x86-64.pkg|

|Ufop环境|下载|
|------------|------|
|Ubuntu14.04 64位|http://download.gna.org/wkhtmltopdf/0.12/0.12.2/wkhtmltox-0.12.2_linux-trusty-amd64.deb|

###方案
使用`wkhtmltoimage`二进制文件直接实现`html`到`图片`的转换，ufop提供转换的参数设置界面和进程调用。

|API 可选参数| 对应`wkhtml2image`指令参数|
|-----------|-------------------------|
|/croph/(>0) |--crop-h|
|/cropw/(>0) |--crop-w|
|/cropx/(>0) |--crop-w|
|/cropy/(>0) |--crop-y|
|/format/[png, jpg] |--format|
|/height/(>0) |--height|
|/quality/(0,100] |--quality 0-100, default 94|
|/width/(>0)| --width|
|/force/[1, 0]|--disable-smart-width|
