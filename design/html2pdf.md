使用wkhtml2pdf组件来将html页面转换为pdf文档

测试：http://download.gna.org/wkhtmltopdf/0.12/0.12.2/wkhtmltox-0.12.2_osx-cocoa-x86-64.pkg
线上：http://download.gna.org/wkhtmltopdf/0.12/0.12.2/wkhtmltox-0.12.2_linux-trusty-amd64.deb

html2pdf/

|可选参数|对应指令参数|
|------|-----------|
|gray/[1 or 0] | -g, --grayscale |
|low/[1 or 0]  | -l, --lowquality |
|orient/[Landscape or Portrait] | -O, --orientation  |
|size/[A4, A3 ...] | -s --page-size  |
|title/<Encoded Title>/ | --title   |
|collate/[1 or 0] | --collate, --no-collate   |
|copies/[1 or more] | --copies   |


Collate copies

Your printer can sort multiple copy jobs. 
For example, if you print two copies of a three-page document and you choose not to collate them, 
the pages print in this order: 1, 1, 2, 2, 3, 3. If you choose to collate, the pages print in this order: 1, 2, 3, 1, 2, 3.