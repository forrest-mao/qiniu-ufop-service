#简介

该命令用来将空间中的图片按照格子模型合成为一个图片。
支持的原图片格式为`png`和`jpeg`，支持的目标图片格式为`png`和`jpeg`。
如果你希望输出的目标图片格式支持其他的格式，或者再对输出的目标图片进行裁剪，缩放，加水印等操作，
可以结合七牛已有的图片处理指令`imageView2`或`imageMogr2`进行管道处理。

#命令

该命令的名称为`imagecomp`，对应的ufop实例名称为`ufop_prefix`+`imagecomp`。

```
imagecomp
/bucket/<string>

/format/<string>
/rows/<int>
/cols/<int>
/halign/<string>
/valign/<string>
/alpha/<int>
/order/<int>
/bgcolor/<string>

/url/<string>
/url/<string>
/url/<string>
....

```

**PS: 该命令的可选参数的指定顺序可以是任意的，必须指定的参数请按照如上所示格式设置。**

#参数

|参数名|描述|可选|
|--------|---------|---------|
|bucket|原图片所在空间名称，指定的值为空间名称经过`Url安全Base64编码`后的值，命令检查后面的url参数对应的文件是否在这个空间中|必须|
|format|合成图片的输出格式，可选值为`png`和`jpeg`，默认为`jpeg`，如果不指定该参数的话|可选|
|rows|原图片的组合方式中，图片排列的行数，如果不指定，则根据`url`的数量和指定的`cols`计算出|可选|
|cols|原图片的组合方式中，图片排列的列数，如果不指定，在`rows`指定的情况下，会根据`url`的数量和`rows`来计算出；如果`rows`也没有指定，则默认为`1`，并反推出`rows`的值|可选|
|halign|在格子模型中，每个图片在自己所在格子里面水平方向的对齐方式，可选值为`left`,`center`和`right`，默认为`left`|可选|
|valign|在格子模型中，每个图片在自己所在格子里面垂直方向的对齐方式，可选值为`top`,`middle`和`bottom`，默认为`top`|可选|
|alpha|合成图片的输出结果的背景透明度，可选值为`[0,255]`，在需要合成的原图片都是透明背景的`png`的情况下，需要输出图片背景透明，请设置为`0`|可选|
|order|需要合成的原图片在目标图片的格子模型中的粘贴顺序，可选值为`0`和`1`；默认为`1`，表示按照列的顺序来粘贴；`0`表示按照行的方式粘贴|可选|
|bgcolor|合成图片的输出结果的背景颜色，指定的格式为`#FFFFFF`，指定的值是对格式如`#FFFFFF`的颜色做`Url安全Base64编码`后的值|可选|
|url|需要合成的原图片的可访问外链，这些图片必须在上面所指定的空间中，至少指定一个图片外链|必须|

备注：
1. 如果`rows`和`cols`都没有指定，那么会生成一个列为`1`的图片。
2. 所谓的格子模型，就像我们把几个图片放在桌上一样，格子模型中的格子长宽，分别是几个图片中最大的长度和最大的宽度，然后其他的图片在这些格子里面摆放。
3. 关于`order`的理解，可以看下面的图片：

**按行粘贴**  
![按行粘贴](images/number_order_by_row.png)

**按列粘贴**  
![按列粘贴](images/number_order_by_col.png)

#配置

由于需要检验指定的`url`确实在指定的`bucket`中，需要配置用户的`AccessKey`和`SecretKey`，这些参数在`imagecomp.conf`里面指定。

|Key|Value|描述|
|----|-----|-------|
|AccessKey|用户的AccessKey，可以在[这里](https://portal.qiniu.com/setting/key)查到|必须设置|
|SecretKey|用户的SecretKey，可以在[这里](https://portal.qiniu.com/setting/key)查到|必须设置|

#创建

如果是初次使用这个ufop的实例，我们需要遵循如下的步骤：

```
创建实例 -> 编译上传镜像 -> 切换镜像版本 -> 生成实例并启动
```

1.使用`qufopctl`的`reg`指令创建`imagecomp`实例，假设前缀为`qntest-`，创建一个私有的ufop实例。

```
$ qufopctl reg qntest-imagecomp -mode=2 -desc='imagecomp ufop'
Ufop name:	 qntest-imagecomp
Access mode:	 PRIVATE
Description:	 imagecomp ufop
```

2.准备ufop的镜像文件。

```
$ tree imagecomp

imagecomp
├── imagecomp.conf
├── qufop
├── qufop.conf
└── ufop.yaml
```

3.使用`qufopctl`的`build`指令构建并上传`imagecomp`实例的项目文件。

```
$ qufopctl build qntest-imagecomp -dir imagecomp
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

4.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
$ qufopctl imageinfo qntest-imagecomp
version: 1
state: building
createAt: 2015-09-26 08:46:01.814350722 +0800 CST
```

5.使用`qufopctl`的`info`来查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-imagecomp
Ufop name:	 qntest-imagecomp
Owner:		 1380435366
Version:	 0
Access mode:	 PRIVATE
Description:
Create time:	 2015-09-26 08:45:47 +0800 CST
Instance num:	 1
Max instanceNum: 5
Flavor:	 default
Access list:	 1380435366
```

如果我们看到的版本号`Version`是`0`的话，说明当前没有运行任何版本的镜像。

6.等待第4步中的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
 $ qufopctl ufopver qntest-imagecomp -c 1
```

7.再次使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-imagecomp
Ufop name:	 qntest-imagecomp
Owner:		 1380435366
Version:	 1
Access mode:	 PRIVATE
Description:
Create time:	 2015-09-26 08:45:47 +0800 CST
Instance num:	 1
Max instanceNum: 5
Flavor:	 default
Access list:	 1380435366
```

8.使用`qufopctl`的`resize`指令来启动`ufop`的实例。

```
$ qufopctl resize qntest-imagecomp -num 1
Resize instance num from 0 to 1.

	instance 1	[state] Running
```

9.然后就可以使用七牛标准的fop使用方式来使用这个`qntest-imagecomp`名称的`ufop`了。

#更新

如果是需要对一个已有的ufop实例更新镜像的版本，我们需要遵循如下的步骤：

```
编译上传镜像 -> 切换镜像版本 -> 更新实例
```

1.使用`qufopctl`的`build`指令构建并上传`imagecomp`实例的项目文件。

```
$ qufopctl build qntest-imagecomp -dir imagecomp
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

2.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
version: 1
state: build success
createAt: 2015-09-26 08:46:01.814350722 +0800 CST

version: 2
state: building
createAt: 2015-09-26 08:57:18.91926743 +0800 CST
```

3.等待第2步中的新的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
 $ qufopctl ufopver qntest-imagecomp -c 2
```

4.更新线上实例的镜像版本。

```
$ qufopctl upgrade qntest-imagecomp
```

5.使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-imagecomp
Ufop name:	 qn-imagecomp
Owner:		 1380435366
Version:	 2
Access mode:	 PRIVATE
Description:
Create time:	 2015-09-26 09:45:47 +0800 CST
Instance num:	 1
Max instanceNum: 5
Flavor:	 default
Access list:	 1380435366
```

#示例

1.我们把九个数字图片合成为一个：

```
qntest-imagecomp
/bucket/cHVibGlj
/format/png
/halign/center
/valign/bottom
/rows/3
/cols/3
/order/0
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMS5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMi5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMy5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNC5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNS5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNi5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNy5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uOC5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uOS5wbmc=
```
结果：

![3x3byrow](images/number_order_by_row.png)

2.我们把九个数字图片排成一列：

```
qntest-imagecomp
/bucket/cHVibGlj
/format/png
/halign/center
/valign/bottom
/cols/1
/order/1
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMS5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMi5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMy5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNC5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNS5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNi5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNy5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uOC5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uOS5wbmc=
```
结果：

![9x1bycol](images/number_in_a_col.png)

3.我们把九个数字图片排成不规则的图片：

```
qntest-imagecomp
/bucket/cHVibGlj
/format/png
/halign/centernum
/valign/bottom
/cols/5
/order/1
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMS5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMi5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uMy5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNC5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNS5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNi5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uNy5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uOC5wbmc=
/url/aHR0cDovLzd4bHQyay5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS9uOS5wbmc=
```

![2x2bycol](images/number_in_a_layout.png)
