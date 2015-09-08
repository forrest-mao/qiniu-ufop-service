#简介
该命令用来将上传到七牛空间中的zip文件进行解压。在某些场景下，用户需要将很多的小文件打包上传以提升上传的效率，上传完之后可以在七牛的空间中解压出一个个文件。该命令实现了zip包的解压功能，并且支持对文件名进行gbk或utf8编码的zip包。也就是说Windows下面使用自带zip工具压缩的文件可以直接上传解压。其他的场景下，可以对文件名进行utf8编码然后打包为zip文件上传，比如移动端（Android或iOS平台）。

#命令
该命令名称为`unzip`，对应的ufop实例名称为`ufop_prefix`+`unzip`。
```
unzip/bucket/<UrlsafeBase64EncodedBucket>/prefix/<UrlsafeBase64EncodedPrefix>/overwrite/<1 or 0>
```

#参数
|参数名|描述|可选|
|----------|------------|---------|
|bucket|解压到指定的空间名称|必填|
|prefix|为解压后的文件名称添加一个前缀|可选，默认为空|
|overwrite|是否覆盖空间中原有的同名文件|可选，默认为0，不覆盖|

**PS: 参数有固定的顺序，可选参数可以不设置**

**备注**：

1. `bucket`参数必须使用UrlsafeBase64编码方式编码。
2. `prefix`参数必须使用UrlsafeBase64编码方式编码。

#配置
出于安全性的考虑，你可以根据实际的需求设置如下参数来控制unzip功能的安全性:

|Key|Value|描述|
|-------|---------|-------------|
|unzip_max_zip_file_length|默认为1GB|zip文件自身的最大大小，单位：字节，这个参数需要严格控制，以避免被恶意利用|
|unzip_max_file_length|默认为100MB|zip文件中打包的单个文件的最大大小，单位：字节，这个参数需要严格控制，以避免被恶意利用|
|unzip_max_file_count|默认为10|zip文件中打包的文件数量，这个参数需要严格控制，以避免被恶意利用|

如果需要自定义，你需要在`qufop.conf`的配置文件中添加这两项。

#创建

```
创建实例 -> 编译上传镜像 -> 切换镜像版本 -> 生成实例并启动
```

1.使用`qufopctl`的`reg`指令创建`unzip`实例，假设前缀为qntest-，创建一个私有的ufop实例。

```
$ qufopctl reg qntest-unzip -mode=2 -desc='unzip ufop'
Ufop name:	 qntest-unzip
Access mode:	 PRIVATE
Description:	 unzip ufop
```

2.准备ufop镜像文件。

```
$ tree unzip
unzip
├── qufop
├── qufop.conf
└── ufop.yaml
```

其中`qufop`是编译好的可执行文件。必须使用`chmod +x qufop`来赋予可执行权限。`qufop.conf`为`qufop`运行需要的配置文件，对于`unzip`功能来讲，它可能有如下的配置信息：

```
{
    "listen_port": 9100,
    "listen_host": "0.0.0.0",
    "read_timeout": 1800,
    "write_timeout": 1800,
    "max_header_bytes": 65535,
    "ufop_prefix":"qntest-",
    "access_key": "TQt-iplt8zbK3LEHMjNYyhh6PzxkbelZFRMl10xx",
    "secret_key": "hTIq4H8N5NfCme8gDvZqr6EDmvlIQsRV5L65bVva",
    "unzip_max_zip_file_length":104857600,
    "unzip_max_file_length":100000,
    "unzip_max_file_count":10
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

3.使用`qufopctl`的`build`指令构建并上传`unzip`实例的项目文件。

```
$ qufopctl build qntest-unzip -dir='unzip'
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

4.使用`qufopctl`的`imageinfo`来查看已上传的镜像。

```
$ qufopctl imageinfo qntest-unzip
version: 1
state: building
createAt: 2015-04-06 22:37:22.360479712 +0800 CST
```

5.使用`qufopctl`的`info`来查看当前ufop所使用的镜像。

$ qufopctl info qntest-unzip
```
Ufop name:	 qntest-unzip
Owner:		 1380340116
Version:	 0
Access mode:	 PRIVATE
Description:	 unzip ufop
Create time:	 1970-01-01 08:00:00 +0800 CST
Instance num:	 0
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```
我们看到`Version`的值为`0`，说明当前没有可用的版本。

6.使用`qufopctl`的`ufopver`指令切换当前ufop所使用的镜像版本。

```
$ qufopctl ufopver qntest-unzip -c=1
```

7.再次使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-unzip
Ufop name:	 qntest-unzip
Owner:		 1380340116
Version:	 1
Access mode:	 PRIVATE
Description:	 unzip ufop
Create time:	 1970-01-01 08:00:00 +0800 CST
Instance num:	 0
Max instanceNum: 5
Flavor:	 default
Access list:	 1380340116
```

8.使用`qufopctl`的`resize`指令来启动`ufop`的实例。

```
$ qufopctl resize qntest-unzip -num=1
Resize instance num from 1 to 1.
```

9.然后就可以使用七牛标准的fop使用方式来使用这个`qntest-unzip`名称的`ufop` 了。

#更新

如果是需要对一个已有的ufop实例更新镜像的版本，我们需要遵循如下的步骤：

```
编译上传镜像 -> 切换镜像版本 -> 更新实例
```

1.使用`qufopctl`的`build`指令构建并上传`unzip`实例的项目文件。

```
$ qufopctl build qntest-unzip -dir unzip
checking files ...
getting upload token ...
making .tar file ...
uploading .tar file ...
upload .tar succeed, please check 'imageinfo' and 'ufopver'.
```

2.使用`qufopctl`的`imageinfo`来查看已经上传的镜像。

```
$ qufopctl imageinfo qntest-unzip
version: 1
state: build success
createAt: 2015-04-06 21:50:50.780011704 +0800 CST

version: 2
state: building
createAt: 2015-09-08 16:39:09.537306064 +0800 CST
```

3.等待第2步中的新的镜像的状态变成`build success`的时候，就可以使用`qufopctl`的`ufopver`指令来切换当前ufop所使用的镜像版本。

```
$ qufopctl ufopver qntest-unzip -c 2
```

4.更新线上实例的镜像版本。

```
$ qufopctl upgrade qntest-unzip
```

5.使用`qufopctl`的`info`指令查看当前ufop所使用的镜像版本。

```
$ qufopctl info qntest-unzip
Ufop name:   qntest-unzip
Owner:       1380340116
Version:     2
Access mode:     PRIVATE
Description:     unzip ufop
Create time:     2015-04-06 21:42:29 +0800 CST
Instance num:    1
Max instanceNum: 5
Flavor:  default
Access list:     1380340116
```

#示例

```
qntest-unzip/bucket/ZHpkcC10ZXN0
```
该指令解压出来的文件自动上传到指定空间中，所以不需要`saveas`指令。
