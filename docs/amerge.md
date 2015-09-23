#简介

该命令用来将两个音频文件进行混音操作，基于ffmpeg实现。

这里的混音操作必须对空间中已有的文件（比如A）进行fop调用，另外指令中所指定的`url`对应的文件就是和这个文件（上面的A）进行音频合并操作的。混音的结果就是你可以同时欣赏两个音频。

#命令

该命令的名称为`amerge`，对应的ufop实例名称为`ufop_prefix`+`amerge`。

```
amerge
/format/<string>
/mime/<string>
/bucket/<string>
/url/<string>
/duration/<string>
```

**该命令参数请按照顺序设置**

#参数

|参数名|描述|备注|
|--------|--------|-----|
|format|目标文件格式，比如mp3|通用性最好的是mp3|
|mime|目标文件的MimeType，比如对于mp3就是`audio/mpeg`|需要UrlsafeBase64编码|
|bucket|混音操作的第二个文件，就是那个需要混音进待处理文件的文件所在空间|需要UrlsafeBase64编码|
|url|混音操作的第二个文件的可访问外链，必须可以根据这个外链下载文件的内容，另外第二个文件必须在上面`bucket`参数所指定的空间内|需要UrlsafeBase64编码|
|duration|可选参数，可选值为和`first`,`shortest`,`longest`，默认为`first`，表示目标文件的时长和哪个文件保持一致|如果参数不设置，采用默认值|

#配置
出于安全性的考虑，你可以根据实际需求设置如下参数来控制`amerge`功能的安全性

|Key|Value|描述|
|------|------|-----|
|amerge_max_first_file_length|默认100MB，单位：字节|这个值主要限制待处理文件的大小，出于服务安全性考虑|
|amerge_max_second_file_length|默认100MB，单位：字节|这个值主要限制需要混音到待处理文件中的文件的大小，出于服务安全性考虑|

#创建

本地带编译镜像文件结构

```
amerge
├── bin
│   └── ffmpeg
├── lib
│   ├── libFLAC.so.8
│   ├── libSDL-1.2.so.0
│   ├── libX11.so.6
│   ├── libXau.so.6
│   ├── libXdmcp.so.6
│   ├── libXext.so.6
│   ├── libasound.so.2
│   ├── libasyncns.so.0
│   ├── libc.so.6
│   ├── libcaca.so.0
│   ├── libdbus-1.so.3
│   ├── libdl.so.2
│   ├── libfdk-aac.so.0
│   ├── libjson-c.so.2
│   ├── libm.so.6
│   ├── libmp3lame.so.0
│   ├── libncursesw.so.5
│   ├── libnsl.so.1
│   ├── libogg.so.0
│   ├── libopus.so.0
│   ├── libpthread.so.0
│   ├── libpulse-simple.so.0
│   ├── libpulse.so.0
│   ├── libpulsecommon-4.0.so
│   ├── libresolv.so.2
│   ├── librt.so.1
│   ├── libslang.so.2
│   ├── libsndfile.so.1
│   ├── libtinfo.so.5
│   ├── libva.so.1
│   ├── libvdpau.so.1
│   ├── libvorbis.so.0
│   ├── libvorbisenc.so.2
│   ├── libwrap.so.0
│   ├── libxcb-shape.so.0
│   ├── libxcb-shm.so.0
│   ├── libxcb-xfixes.so.0
│   ├── libxcb.so.1
│   └── libz.so.1
├── qufop
├── amerge.conf
├── qufop.conf
└── ufop.yaml
```

其他镜像编译，部署过程请参考其他命令。
