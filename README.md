## 编译与打包脚本说明：
	build.bat：windows下编译脚本
	build.sh：linux编译脚本，支持打包，打包命令：build.sh pkg

## 工程说明：
```
目录说明：
	1.control目录：存放工程启动调度go文件
	2.etc目录：存放工程配置文件，conf.ini
	3.handlers目录：存放路由处理go文件
	4.dto目录：存放数据模型go文件
	5.service：存放业务处理go文件
	6.modules目录：存放业务数据访问go文件
	7.constant目录：存放常量数据go文件
	8.log目录：存放系统运行日志
	9.common: 公共方法
	10.core：读取配置的方法
	11.vender：三方包

文件说明：
	1.main.go文件：工程主入口文件
	2.main_control.go文件：工程主调度文件
	3.build.sh文件：linux下工程编译脚本
	4.build.bat文件：window下工程编译脚本
	5.conf.yaml文件：工程默认配置文件
```

## 项目结构:
```
项目开发目录：
|--report_api
	|--main.go
	|--control
    |--handler
    |--service
					 	 
```
