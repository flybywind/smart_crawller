# 用Golang写的简单抓取器

## 目标
这不是一个通用的spider，它的目标很单一，即：
通过配置实现对种子站点上多媒体目标的抓取，如图片、声音等。
支持对不同站点配置不同的抓取模板，对文件添加说明文本。

## 设计
蜘蛛把所有网站都定义为3层结构，即：
* 着陆页（landing_page）：即包含着目标（图片、声音）资源的页面. 蜘蛛爬到这里，把资源下载完成，就算结束了。同时在着陆页上，可以提取到说明信息，即对资源的分类说明。如图片种类等
* 容器页（container_page）：即着陆页的父页，其中包含很多子频道，每个子频道就是着陆页。
* root页：即用户指定的其中一个种子，是容器页的直接父节点
* 另外，还要设计一个try dive功能，因为着陆页上的图片多是缩略图，try dive告诉蜘蛛进到着陆页内部，抓取高质图片
