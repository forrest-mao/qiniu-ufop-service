#简介
该命令用来将空间中的html文档转换为pdf文档。

#命令
该命令的名称为`html2pdf`，对应的ufop实例名称为`ufop_prefix`+`html2pdf`。

```
html2pdf
/gray/<int>
/low/<int>
/orient/<string>
/size/<string>
/title/<string>
/collate/<int>
/copies/<int>
```

**PS: 该命令的所有参数都是可选参数，另外参数没有固定顺序。**

#参数

|参数名|描述|可选|
|--------|----------|----------|
|gray|目标PDF文件是否使用黑白颜色，可选值`1`或`0`，如果选择黑白，那么生成的PDF文件里面内容都是黑白的|可选|
|low|目标PDF文件是否选择使用低质量，可选值`1`或`0`，如果选择低质量，那么生成的PDF文件大小会比较小|可选|
|orient|目标PDF文件的方向，可选值为`Landscape`和`Portrait`，默认为`Portrait`|可选|
|size|目标PDF文件的纸张大小，可选值为`A1-A8`或`B1-B8`，默认为`A4`|可选|
|title|目标PDF文件属性中的标题，如果指定的话，必须是对字符串进行`Urlsafe Base64编码`后的值|可选|
|collate|目标PDF文件的多副本打印方式，可选值`1`或`0`，默认为`1`，即采用`collate`模式|可选|
|copies|目标PDF文件的副本数量，默认值为`1`|可选|

**关于`collate`参数的含义：**

这个参数在`copies > 1`的情况下，表现出不同文件打印方式。
举个例子，当你有一个PDF文档，该文件有3页，现在需要输出两份。
当`collate/1`的情况下，输出顺序为`1,2,3,1,2,3`；
当`collate/0`的情况下，输出顺序为`1,1,2,2,3,3`；
默认不指定`collate`的情况下，`collate`为1。


#配置

出于安全性的考虑，你可以根据实际需求设置如下参数来控制`html2pdf`功能的安全性：

|Key|Value|描述|
|------------|-----------|-------------|
|html2pdf_max_page_size|默认为10MB，单位：字节|允许进行文档转换的单个页面的大小|
|html2pdf_max_copies|默认为1|允许输出的PDF文档的最大副本数量|

#创建

如果是初次使用这个ufop的实例，我们需要遵循如下的步骤：

```
创建实例 -> 编译上传镜像 -> 切换镜像版本 -> 生成实例并启动
```

1.使用`qufopctl`的`reg`指令创建`html2pdf`实例，假设前缀为`qntest-`，创建一个私有的ufop实例。

```
$ qufopctl reg qntest-html2pdf -mode=2 -desc='html2pdf ufop'
Ufop name:	 qntest-html2pdf
Access mode:	 PRIVATE
Description:	 html2pdf ufop
```

2.准备ufop的镜像文件。

```
$ tree html2pdf

html2pdf
├── fonts
│   ├── simfang.ttf
│   ├── simhei.ttf
│   ├── simkai.ttf
│   ├── simpbdo.ttf
│   ├── simpfxo.ttf
│   ├── simpo.ttf
│   ├── simsun.ttc
│   └── simsunb.ttf
├── html2pdf.conf
├── pkg
│   └── wkhtmltox-0.12.2_linux-trusty-amd64.deb
├── qufop
└── ufop.yaml
```

其中`fonts`目录下面为支持中文的字体，`pkg`下面为`xkhtmltopdf`的二进制可执行文件。

3.使用`qufopctl`的`build`指令构建并上传`html2pdf`实例的项目文件。

```
$ qufopctl build qntest-html2pdf -dir html2pdf
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

4.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
$ qufopctl imageinfo qntest-html2pdf
version: 1
state: building
createAt: 2015-09-08 15:33:15.132727309 +0800 CST
```

5.使用`qufopctl`的`info`来查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-html2pdf
Ufop name:	 qntest-html2pdf
Owner:		 1380340116
Version:	 0
Access mode:	 PRIVATE
Description:	 html2pdf ufop
Create time:	 2015-09-08 15:29:18 +0800 CST
Instance num:	 0
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

如果我们看到的版本号`Version`是`0`的话，说明当前没有运行任何版本的镜像。

6.等待第4步中的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
 $ qufopctl ufopver qntest-html2pdf -c 1
```

7.再次使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-html2pdf
Ufop name:	 qntest-html2pdf
Owner:		 1380340116
Version:	 1
Access mode:	 PRIVATE
Description:	 html2pdf ufop
Create time:	 2015-09-08 15:29:18 +0800 CST
Instance num:	 0
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

8.使用`qufopctl`的`resize`指令来启动`ufop`的实例。

```
$ qufopctl resize qntest-html2pdf -num 1
Resize instance num from 0 to 1.

	instance 1	[state] Running
```

9.然后就可以使用七牛标准的fop使用方式来使用这个`qntest-html2pdf`名称的`ufop`了。

#更新

如果是需要对一个已有的ufop实例更新镜像的版本，我们需要遵循如下的步骤：

```
编译上传镜像 -> 切换镜像版本 -> 更新实例
```

1.使用`qufopctl`的`build`指令构建并上传`html2pdf`实例的项目文件。

```
$ qufopctl build qntest-html2pdf -dir html2pdf
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

2.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
$ qufopctl imageinfo qntest-html2pdf
version: 1
state: build success
createAt: 2015-09-08 15:33:15.132727309 +0800 CST

version: 2
state: building
createAt: 2015-09-08 15:47:11.179527356 +0800 CST
```

3.等待第2步中的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
 $ qufopctl ufopver qntest-html2pdf -c 2
```

4.更新线上实例的镜像版本。

```
$ qufopctl upgrade qntest-html2pdf
```

5.使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-html2pdf
Ufop name:	 qntest-html2pdf
Owner:		 1380340116
Version:	 1
Access mode:	 PRIVATE
Description:	 html2pdf ufop
Create time:	 2015-09-08 15:29:18 +0800 CST
Instance num:	 1
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

#示例


```
qntest-html2pdf
```

```
qntest-html2pdf/orient/size/A4/low/1
```

持久化的使用方式

```
qntest-html2pdf/orient/size/A4/low/1|saveas/aWYtcGJsOnRlc3QucGRm
```

其中`aWYtcGJsOnRlc3QucGRm`为目标存储空间和目标图片PDF文件的`Urlsafe Base64编码`。

