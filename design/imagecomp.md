图片合成

imagecomp
/bucket/<string>
/format/<string> 	optional, default jpg
/halign/<string> 	optional, default left
/valign/<string> 	optional, default top
/row/<int>			optional, default 1
/col/<int>			optional, default 1
/order/<int>		optional, default 0
/alpha/<int> 		optional, default 0
/bgcolor/<string>	optional, default gray
/url/<string>		
/url/<string>
...


halign取值

left, right, center

valign取值 

top, bottom, middle


halign default = left
valign default = top


order取值

0 表示从行开始依次粘贴(默认)
1 表示从列开始依次粘贴

提供的row和col的值必须能够和urls的数量匹配起来，最好是 row * col = len(urls)
如果提供的 len(urls) > row * col 直接报错，url count larger than row * col;
如果提供的 len(urls) < row * col ，则按照如下情况处理：
1. 如果 order ＝ 0，即按照row的方式排列，那么如果 len(urls) < (row -1)* col 即报错
2. 如果 order ＝ 1，即按照col的方式排列，那么如果 len(urls) < row * (col - 1) 即报错