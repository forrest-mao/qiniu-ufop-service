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

|参数名|描述|备注|
|--------|--------|-----|
|format|目标文件格式，比如mp3|通用性最好的是mp3|
|mime|目标文件的MimeType，比如对于mp3就是`audio/mpeg`|需要UrlsafeBase64编码|
|bucket|混音操作的第二个文件，就是那个需要混音进待处理文件的文件所在空间|需要UrlsafeBase64编码|
|url|混音操作的第二个文件的可访问外链，必须可以根据这个外链下载文件的内容，另外第二个文件必须在上面`bucket`参数所指定的空间内|需要UrlsafeBase64编码|
|duration|可选参数，可选值为和`first`,`shortest`,`longest`，默认为`first`，表示目标文件的时长和哪个文件保持一致|如果参数不设置，采用默认值|
