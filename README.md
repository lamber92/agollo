Agollo - Go Client for Apollo
================

[![golang](https://img.shields.io/badge/Language-Go-green.svg?style=flat)](https://golang.org)
[![Build Status](https://github.com/apolloconfig/agollo/actions/workflows/go.yml/badge.svg)](https://github.com/apolloconfig/agollo/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/apolloconfig/agollo)](https://goreportcard.com/report/github.com/apolloconfig/agollo)
[![codebeat badge](https://codebeat.co/badges/bc2009d6-84f1-4f11-803e-fc571a12a1c0)](https://codebeat.co/projects/github-com-apolloconfig-agollo-master)
[![Coverage Status](https://coveralls.io/repos/github/apolloconfig/agollo/badge.svg?branch=master)](https://coveralls.io/github/apolloconfig/agollo?branch=master)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoDoc](http://godoc.org/github.com/apolloconfig/agollo?status.svg)](http://godoc.org/github.com/apolloconfig/agollo)
[![GitHub release](https://img.shields.io/github/release/apolloconfig/agollo.svg)](https://github.com/apolloconfig/apolloconfig/releases)
[![996.icu](https://img.shields.io/badge/link-996.icu-red.svg)](https://996.icu)

方便Golang接入配置中心框架 [Apollo](https://github.com/ctripcorp/apollo) 所开发的Golang版本客户端。

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
go get -u github.com/apolloconfig/agollo/v4@latest
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
		AppID:          "testApplication_yang",
		Cluster:        "dev",
		IP:             "http://106.54.227.205:8080",
		NamespaceName:  "dubbo",
		IsBackupConfig: true,
		Secret:         "6ce3ff7e96a24335a9634fe9abca6d51",
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

***使用Demo*** ：[agollo_demo](https://github.com/zouyx/agollo_demo)

***其他语言*** ： [agollo-agent](https://github.com/zouyx/agollo-agent.git) 做本地agent接入，如：PHP

欢迎查阅 [Wiki](https://github.com/apolloconfig/agollo/wiki) 或者 [godoc](http://godoc.org/github.com/zouyx/agollo) 获取更多有用的信息

如果你觉得该工具还不错或者有问题，一定要让我知道，可以发邮件或者[留言](https://github.com/apolloconfig/agollo/issues)。

# User

* [使用者名单](https://github.com/apolloconfig/agollo/issues/20)

# Contribution

* Source Code: https://github.com/apolloconfig/agollo/
* Issue Tracker: https://github.com/apolloconfig/agollo/issues

# License

The project is licensed under the [Apache 2 license](https://github.com/apolloconfig/agollo/blob/master/LICENSE).

# Reference

Apollo : https://github.com/ctripcorp/apollo
