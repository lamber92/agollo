Agollo - Go Client for Apollo
================

方便Golang接入配置中心框架 [Apollo](https://github.com/ctripcorp/apollo) 所开发的Golang版本客户端。

基于（[Agollo](https://github.com/apolloconfig/apollo)）项目，结合个人项目做适配修改，不保证同步原代码库逻辑频率。

# Changes

基于原版代码(代码同步到原库v4.3.1)，有以下修改项： 

* 首次连接Apollo服务时检查IP地址、App秘钥合法性，并同步返回连接失败原因；
* 修改Log接口定义，使之与其他开源库Log库更加贴合；
* 修复请求失败后，因为触发重试机制导致body没有释放而可能导致内存泄露的问题；
* 支持动态新增指定namespace配置缓存，并自动监听对应namespace配置变动；
  * 改写 GetConfigAndInit() 方法
  * 改写 notify 包逻辑
* 优化针对请求 notifications/v2/ 接口因为系统架构中存在中间网络组件导致返回http-status-code=401时的处理方式（[#3652](https://github.com/apolloconfig/apollo/issues/3652)）；
* 增加3个常用错误，方便业务层面处理；
* 修改部分变量名称及注释，使之更贴合实际意义；

# Features

* 支持多 IP、AppID、namespace
* 实时同步配置
* 灰度配置
* 延迟加载（运行时）namespace
* 客户端，配置文件容灾
* 自定义日志，缓存组件
* 支持配置访问秘钥

# Usage

## 快速入门

### 导入 agollo

```
replace github.com/apolloconfig/agollo/v4 => github.com/lamber92/agollo/v4 v4.3.2

require (
	github.com/apolloconfig/agollo/v4 v4.3.1
	......
)
```

### 启动 agollo

```
package main

import (
	"fmt"
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
)

func main() {
	c := &config.AppConfig{
		AppID:          "xxxx",
		Cluster:        "dev",
		IP:             "http://xxxx:8080",
		NamespaceName:  "xxxx",
		IsBackupConfig: true,
		Secret:         "xxxx",
	}

	client, _ := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return c, nil
	})
	fmt.Println("初始化Apollo配置成功")

	//Use your apollo key to test
	cache := client.GetConfigCache(c.NamespaceName)
	value, _ := cache.Get("key")
	fmt.Println(value)
}
```

## 更多用法

- 请参考原项目仓库说明：[Agollo](https://github.com/apolloconfig/apollo)
