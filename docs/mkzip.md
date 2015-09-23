#简介
该命令用来创建指定编码方式的zip归档文件。七牛支持的[mkzip功能](http://developer.qiniu.com/docs/v6/api/reference/fop/mkzip.html)默认当前仅支持utf8编码方式，该编码方式打包的文件在Windows操作系统下面使用系统自带的unzip功能时，会造成中文文件名称乱码。该命令通过指定文件名称编码为gbk的方式可以解决这个问题。目前支持utf8（默认）和gbk（手动指定）两种编码方式。

**备注**：该命令只能对指定空间中的文件进行打包操作，支持的最大文件数量为1000。

#命令
该命令名称为`mkzip`，对应的ufop实例名称为`ufop_prefix`+`mkzip`。

```
mkzip
/bucket/<UrlsafeBase64EncodedBucket>
/encoding/<UrlsafeBase64EncodedEncoding>
/url/<UrlsafeBase64EncodedURL>/alias/<UrlsafeBase64EncodedAlias>
/url/<UrlsafeBase64EncodedURL>/alias/<UrlsafeBase64EncodedAlias>
...
```

**PS: 参数有固定的顺序，可选参数可以不设置**

#参数
|参数名|描述|可选|
|-------|---------|-----------|
|bucket|需要打包的文件所在的空间名称|必须|
|encoding|需要打包的文件名称的编码，支持gbk和utf8，默认为utf8|可选|
|url|需要打包的文件可访问的链接，必须存在于`bucket`中|至少指定一个链接|
|alias|需要打包的文件所对应的别名，和`url`配对使用|可以不设置|

**备注**：所有的的参数必须使用`UrlsafeBase64`编码方式编码。

#配置
出于安全性的考虑，你可以根据实际的需求设置如下参数来控制mkzip功能的安全性：

|Key|Value|描述|
|--------|------------|----------------|
|mkzip_max_file_length|默认为100MB，单位：字节|允许打包的文件的单个文件最大字节长度|
|mkzip_max_file_count|默认为100个|允许打包的文件的最大总数量，最多支持1000|

如果需要自定义，你需要在`qufop.conf`的配置文件中添加这两项。

#常见错误

|错误信息|描述|
|-------|------|
|invalid mkzip command format|发送的ufop的指令格式不正确，请参考上面的命令格式设置正确的指令|
|invalid mkzip paramter 'bucket'|指定的`bucket`参数不正确，必须是对原空间名称进行`urlsafe base64`编码后的值|
|invalid mkzip parameter 'encoding'|指定的`encoding`参数不正确，必须是对原编码名称进行`urlsafe base64`编码后的值|
|invalid mkzip parameter 'url'|指定的`url`列表中有一个不正确，必须是对资源链接进行`urlsafe base64`编码后的值|
|invalid mkzip parameter 'alias'|指定的`alias`列表中有一个不正确，必须是对文件别名进行`urlsafe base64`编码后的值|
|mkzip parameter 'url' format error|指定的`url`列表中有一个不正确，必须是正确的资源链接|
|invalid mkzip resource url|指定的`url`列表中有一个不正确，必须是正确的资源链接|
|duplicate mkzip resource alias|指定的`alias`列表中的别名有重复|
|zip file count exceeds the limit|需要压缩的文件数量超过了ufop的最大值限制，这个最大值在`mkzip.conf`里面设置|
|only support items less than 1000|需要压缩的文件数量超过了ufop的最大限制，目前代码最大允许1000个文件压缩|

#创建

如果是初次使用这个ufop的实例，我们需要遵循如下的步骤：

```
创建实例 -> 编译上传镜像 -> 切换镜像版本 -> 生成实例并启动
```

1.使用`qufopctl`的`reg`指令创建`mkzip`实例，假设前缀为`qntest-`，创建一个私有的ufop实例。

```
$qufopctl reg qntest-mkzip -mode=2 -desc='mkzip ufop'
Ufop name:	 qntest-mkzip
Access mode:	 PRIVATE
Description:	 mkzip ufop
```

2.准备ufop镜像文件。

```
$ tree mkzip

mkzip
├── qufop
├── qufop.conf
├── mkzip.conf
└── ufop.yaml
```

其中`qufop`是编译好的可执行文件。必须使用`chmod +x qufop`来赋予可执行权限。`qufop.conf`为`qufop`运行需要的配置文件，对于`mkzip`功能来讲，它可能有如下的配置信息：

**qufop.conf**

```
{
    "listen_port": 9100,
    "listen_host": "0.0.0.0",
    "read_timeout": 1800,
    "write_timeout": 1800,
    "max_header_bytes": 65535,
    "ufop_prefix":"qntest-"
}
```

**mkzip.conf**

```
{
    "access_key": "TQt-iplt8zbK3LEHMjNYyhh6PzxkbelZFRMl10xx",
    "secret_key": "hTIq4H8N5NfCme8gDvZqr6EDmvlIQsRV5L65bVva",
    "mkzip_max_file_length":104857600,
    "mkzip_max_file_count":20
}
```

注意配置文件里面`ufop_prefix`和注册的ufop名称前缀一致。

`ufop.yaml`是七牛ufop规范所要求的镜像构建配置文件，内容如下：

```
image: ubuntu
build_script:
 - echo building...
 - mv $RESOURCE/* .
run: ./qufop qufop.conf
```

3.使用`qufopctl`的`build`指令构建并上传`mkzip`实例的项目文件。

```
$ qufopctl build qntest-mkzip -dir mkzip
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

4.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
$ qufopctl imageinfo qntest-mkzip
version: 1
state: building
createAt: 2015-04-06 21:50:50.780011704 +0800 CST
```

5.使用`qufopctl`的`info`来查看当前ufop所使用的镜像。

```
$ qufopctl info qntest-mkzip
Ufop name:	 qntest-mkzip
Owner:		 1380340116
Version:	 0
Access mode:	 PRIVATE
Description:	 mkzip ufop
Create time:	 1970-01-01 08:00:00 +0800 CST
Instance num:	 1
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

我们看到`Version`的值为`0`，说明当前没有可用的版本。

6.使用`qufopctl`的`ufopver`指令切换当前ufop所使用的镜像版本。

```
$ qufopctl ufopver qntest-mkzip -c 1
```

7.再次使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-mkzip
Ufop name:	 qntest-mkzip
Owner:		 1380340116
Version:	 1
Access mode:	 PRIVATE
Description:	 mkzip ufop
Create time:	 1970-01-01 08:00:00 +0800 CST
Instance num:	 1
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

8.使用`qufopctl`的`resize`指令来启动`ufop`的实例。

```
$ qufopctl resize qntest-mkzip -num 1
Resize instance num from 1 to 1.
```

9.然后就可以使用七牛标准的fop使用方式来使用这个`qntest-mkzip`名称的`ufop` 了。

#更新

如果是需要对一个已有的ufop实例更新镜像的版本，我们需要遵循如下的步骤：

```
编译上传镜像 -> 切换镜像版本 -> 更新实例
```

1.使用`qufopctl`的`build`指令构建并上传`mkzip`实例的项目文件。

```
$ qufopctl build qntest-mkzip -dir mkzip
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

2.使用`qufopctl`的`imageinfo`来查看已经上传的镜像。

```
$ qufopctl imageinfo qntest-mkzip
version: 1
state: build success
createAt: 2015-04-06 21:50:50.780011704 +0800 CST

version: 2
state: building
createAt: 2015-09-08 16:39:09.537306064 +0800 CST
```

3.等待第2步中的新的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
$ qufopctl ufopver qntest-mkzip -c 2
```

4.更新线上实例的镜像版本。

```
$ qufopctl upgrade qntest-mkzip
```

5.使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-mkzip
Ufop name:   qntest-mkzip
Owner:       1380340116
Version:     2
Access mode:     PRIVATE
Description:     mkzip ufop
Create time:     2015-04-06 21:42:29 +0800 CST
Instance num:    1
Max instanceNum: 5
Flavor:  default
Access list:     1380340116
```

#示例

持久化的使用方式：

```
qntest-mkzip
/bucket/aWYtcGJs
/encoding/Z2Jr
/url/aHR0cDovLzdwbjY0Yy5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS8yMDE1LzAzLzIyL3Fpbml1Lm1wNA==/alias/5LiD54mb5a6j5Lyg54mH
/url/aHR0cDovLzdwbjY0Yy5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS8yMDE1LzAzLzIyL3Fpbml1LnBuZw==
/url/aHR0cDovLzdwbjY0Yy5jb20xLnowLmdsYi5jbG91ZGRuLmNvbS8yMDE1LzAzLzI3LzEzLmpwZw==/alias/MjAxNS9waG90by5qcGc=
|saveas/aWYtcGJsOnFpbml1LnppcA==
```

上面的写法是格式化后便于理解，实际使用中没有换行符号。

其中`saveas`的参数为保存的目标空间和目标文件名的`Urlsafe Base64编码`。
