#简介
该命令用来将空间中的html文档转换为图片，支持的目标图片格式为PNG和JPEG两种格式。

#命令
该命令的名称为`html2image`，对应的ufop实例名称为`ufop_prefix`+`html2image`。

```
html2image
/croph/<int>
/cropw/<int>
/cropx/<int>
/cropy/<int>
/format>/<string>
/height/<int>
/width/<int>
/quality/<int>
/force/<int>
```

**PS: 该命令的所有参数都是可选参数，另外参数没有固定顺序。**

#参数

|参数名|描述|可选|
|--------|----------|----------|
|croph|指定裁减后的目标图片的高度，图片下方的部分可能被裁减掉|可选|
|cropw|指定裁减后的目标图片的宽度，图片右方的部分可能被裁减掉|可选|
|cropx|沿X轴的方向，裁减目标图片，从图片左边减去指定的像素|可选|
|cropy|沿Y轴的方向，裁减目标图片，从图片上方减去指定的像素|可选|
|format|目标图片格式，支持png和jpeg两种格式，默认为jpeg格式|可选|
|height|目标图片的高度，单位像素|可选|
|width|目标图片的宽度，单位像素|可选|
|quality|目标图片的质量，可选值[1,100]，默认94|可选|
|force|是否强制目标图片的宽度为指定的宽度，可选值1或0，默认为0，如果设置为1，则目标图片宽度强制为指定值，不合适的宽度设定可能造成图片变形|可选|

#配置

出于安全性的考虑，你可以根据实际需求设置如下参数来控制`html2image`功能的安全性：

|Key|Value|描述|
|------------|-----------|-------------|
|html2image_max_page_size|默认为10MB，单位：字节|允许进行文档转换的单个页面的大小|

#创建

如果是初次使用这个ufop的实例，我们需要遵循如下的步骤：

```
创建实例 -> 编译上传镜像 -> 切换镜像版本 -> 生成实例并启动
```

1.使用`qufopctl`的`reg`指令创建`html2image`实例，假设前缀为`qntest-`，创建一个私有的ufop实例。

```
$ qufopctl reg qntest-html2image -mode=2 -desc='html2image ufop'
Ufop name:	 qntest-html2image
Access mode:	 PRIVATE
Description:	 html2image ufop
```

2.准备ufop的镜像文件。

```
$ tree html2image

html2image
├── fonts
│   ├── simfang.ttf
│   ├── simhei.ttf
│   ├── simkai.ttf
│   ├── simpbdo.ttf
│   ├── simpfxo.ttf
│   ├── simpo.ttf
│   ├── simsun.ttc
│   └── simsunb.ttf
├── html2image.conf
├── pkg
│   └── wkhtmltox-0.12.2_linux-trusty-amd64.deb
├── qufop
└── ufop.yaml
```

其中`fonts`目录下面为支持中文的字体，`pkg`下面为`xkhtmltoimage`的二进制可执行文件。

3.使用`qufopctl`的`build`指令构建并上传`html2image`实例的项目文件。

```
$ qufopctl build qntest-html2image -dir html2image
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

4.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
$ qufopctl imageinfo qntest-html2image
version: 1
state: building
createAt: 2015-09-08 15:33:15.132727309 +0800 CST
```

5.使用`qufopctl`的`info`来查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-html2image
Ufop name:	 qntest-html2image
Owner:		 1380340116
Version:	 0
Access mode:	 PRIVATE
Description:	 html2image ufop
Create time:	 2015-09-08 15:29:18 +0800 CST
Instance num:	 0
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

如果我们看到的版本号`Version`是`0`的话，说明当前没有运行任何版本的镜像。

6.等待第4步中的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
 $ qufopctl ufopver qntest-html2image -c 1
```

7.再次使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-html2image
Ufop name:	 qntest-html2image
Owner:		 1380340116
Version:	 1
Access mode:	 PRIVATE
Description:	 html2image ufop
Create time:	 2015-09-08 15:29:18 +0800 CST
Instance num:	 0
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

8.使用`qufopctl`的`resize`指令来启动`ufop`的实例。

```
$ qufopctl resize qntest-html2image -num 1
Resize instance num from 0 to 1.

	instance 1	[state] Running
```

9.然后就可以使用七牛标准的fop使用方式来使用这个`qntest-html2image`名称的`ufop`了。

#更新

如果是需要对一个已有的ufop实例更新镜像的版本，我们需要遵循如下的步骤：

```
编译上传镜像 -> 切换镜像版本 -> 更新实例
```

1.使用`qufopctl`的`build`指令构建并上传`html2image`实例的项目文件。

```
$ qufopctl build qntest-html2image -dir html2image
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

2.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
$ qufopctl imageinfo qntest-html2image
version: 1
state: build success
createAt: 2015-09-08 15:33:15.132727309 +0800 CST

version: 2
state: building
createAt: 2015-09-08 15:47:11.179527356 +0800 CST
```

6.等待第2步中的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
 $ qufopctl ufopver qntest-html2image -c 2
```

7.更新线上实例的镜像版本。

```
$ qufopctl upgrade qntest-html2image
```

8.使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-html2image
Ufop name:	 qntest-html2image
Owner:		 1380340116
Version:	 1
Access mode:	 PRIVATE
Description:	 html2image ufop
Create time:	 2015-09-08 15:29:18 +0800 CST
Instance num:	 1
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

#示例


```
qntest-html2image
```

```
qntest-html2image/format/png/width/100
```

持久化的使用方式

```
qntest-html2image/format/png/width/100|saveas/aWYtcGJsOnRlc3QucG5n
```

其中`aWYtcGJsOnRlc3QucG5n`为目标存储空间和目标图片文件名的`Urlsafe Base64编码`。

